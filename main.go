package main

import (
	"encoding/json"
	"github.com/bwmarrin/lit"
	"github.com/kkyr/fig"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const configFile = "config.yml"
const baseAPIUrl = "https://api.cloudflare.com/client/v4/zones/"

var (
	records []zoneAndRecords
	cfg     config
)

func init() {
	lit.Info("\nStarting up")
	// Loads the config file
	checkErr(fig.Load(&cfg, fig.File(configFile)))
	http.DefaultClient.Timeout = time.Second * 3
	getRecords()
	if strings.ToLower(cfg.LogLevel) == "info" {
		lit.LogLevel = lit.LogInformational
	}
	lit.Info("\nFinished initial setup")
}

func main() {
	wg := sync.WaitGroup{}
	var (
		currIPv4 string
		currIPv6 string
		newIPv4  string
		newIPv6  string
	)
	for {
		lit.Info("\nChecking public IP updates")
		wg.Add(2)
		go getIP("https://api.ipify.org", &newIPv4, &wg)
		go getIP("https://api6.ipify.org", &newIPv6, &wg)
		wg.Wait()
		lit.Info("\npublic IP check complete")
		if currIPv4 != newIPv4 {
			lit.Info("\ndetected public IPv4 change.%s setting to ", currIPv4, newIPv4)
			currIPv4 = newIPv4
			for _, zone := range records {
				zone := zone
				for _, record := range zone.Records {
					record := record
					if record.Type == "A" {
						wg.Add(1)
						go patchRecord(zone, record, currIPv4, &wg)
					}
				}
			}
		}
		if currIPv6 != newIPv6 {
			lit.Info("\ndetected public IPv6 change.%s setting to %s", currIPv6, newIPv6)
			currIPv6 = newIPv6
			for _, zone := range records {
				zone := zone
				for _, record := range zone.Records {
					record := record
					if record.Type == "AAAA" {
						wg.Add(1)
						go patchRecord(zone, record, currIPv6, &wg)
					}
				}
			}
		}
		wg.Wait()
		lit.Info("\nUpdate cycle complete")
		time.Sleep(cfg.Timeout)
	}
}

func checkErr(err error) {
	if err != nil {
		lit.Error("\n%s", err)
		os.Exit(1)
	}
}

// return public IP address of the machine
func getIP(url string, ip *string, group *sync.WaitGroup) {
	defer group.Done()
	response, err := http.Get(url)
	if err == nil {
		defer func(Body io.ReadCloser) {
			checkErr(Body.Close())
		}(response.Body)
		out, err := io.ReadAll(response.Body)
		if err != nil {
			lit.Error("\n%s", err)
		}
		*ip = string(out)
	}
}

// return A and AAAA records hosted on cloudflare
func getRecords() {
	request, err := http.NewRequest("GET", baseAPIUrl, nil)
	request.Header.Add("authorization", "Bearer "+cfg.Token)
	checkErr(err)
	response, err := http.DefaultClient.Do(request)
	checkErr(err)
	defer func(Body io.ReadCloser) {
		checkErr(response.Body.Close())
	}(response.Body)
	var zones apiZones
	checkErr(json.NewDecoder(response.Body).Decode(&zones))
	if zones.Success {
		wg := sync.WaitGroup{}
		defer wg.Wait()
		for i, zone := range zones.Result {
			if allowedRecords, ok := cfg.Domains[zone.Name]; ok {
				wg.Add(1)
				records = append(records, zoneAndRecords{
					ZoneID:  zone.ID,
					Records: nil,
				})
				v4mutex := sync.Mutex{}
				v6mutex := sync.Mutex{}
				i, zone := i, zone
				go func() {
					defer wg.Done()
					request, err := http.NewRequest("GET", strings.Join([]string{baseAPIUrl, zone.ID, "/dns_records/"}, ""), nil)
					request.Header.Add("authorization", "Bearer "+cfg.Token)
					checkErr(err)
					response, err := http.DefaultClient.Do(request)
					checkErr(err)
					var apiResponse apiRecords
					checkErr(json.NewDecoder(response.Body).Decode(&apiResponse))
					checkErr(response.Body.Close())
					if apiResponse.Success {
						for _, record := range apiResponse.Result {
							if record.Type == "A" {
								if _, ok := allowedRecords.V4Records[record.Name]; ok {
									v4mutex.Lock()
									records[i].Records = append(records[i].Records, record)
									v4mutex.Unlock()
								}
							} else if record.Type == "AAAA" {
								if _, ok := allowedRecords.V6Records[record.Name]; ok {
									v6mutex.Lock()
									records[i].Records = append(records[i].Records, record)
									v6mutex.Unlock()
								}
							}
						}
					}
				}()
			}
		}
	}
}

func patchRecord(zone zoneAndRecords, record dnsRecord, ip string, wg *sync.WaitGroup) {
	defer wg.Done()
	request, err := http.NewRequest("PATCH", strings.Join([]string{baseAPIUrl, zone.ZoneID, "/dns_records/", record.ID}, ""),
		strings.NewReader("{\"content\":\""+ip+"\"}"))
	request.Header.Add("authorization", "Bearer "+cfg.Token)
	request.Header.Add("content-type", "application/json")
	checkErr(err)
	response, err := http.DefaultClient.Do(request)
	checkErr(err)
	defer func(Body io.ReadCloser) {
		checkErr(response.Body.Close())
	}(response.Body)
	var apiResponse apiFeedback
	checkErr(json.NewDecoder(response.Body).Decode(&apiResponse))
	if !apiResponse.Success {
		lit.Error("\n%s", apiResponse.Errors)
	}
}
