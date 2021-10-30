package main

import (
	"encoding/json"
	"github.com/kkyr/fig"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const CONFIG_FILE = "config.yml"
const BASE_API_URL = "https://api.cloudflare.com/client/v4/zones/"

var (
	records  []zoneAndRecords
	cfg		Config
)

func init() {
	log.Print("Starting up")
	// Loads the config file
	checkErr(fig.Load(&cfg, fig.File(CONFIG_FILE)))
	http.DefaultClient.Timeout = time.Second * 3
	getRecords()
	log.Print("Finished initial setup")
}

func main() {
	wg := sync.WaitGroup{}
	var (
		currIPv4 string = ""
		currIPv6 string = ""
		newIPv4 string
		newIPv6 string
	)
	for {
		log.Print("Checking public IP updates")
		wg.Add(2)
		go getIP("https://api.ipify.org", &newIPv4, &wg)
		go getIP("https://api6.ipify.org", &newIPv6, &wg)
		wg.Wait()
		log.Print("public IP check complete")
		if currIPv4 != newIPv4 {
			log.Print("detected public IPv4 change.", currIPv4, " setting to ", newIPv4)
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
			log.Print("detected public IPv6 change.", currIPv6, " setting to ", newIPv6)
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
		log.Print("Update cycle complete")
		time.Sleep(cfg.Timeout)
	}
}

func checkErr(err error) {
	if err != nil {
		log.Fatalln(err)
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
			log.Fatalln(err)
		}
		*ip = string(out)
	}
}

// return A and AAAA records hosted on cloudflare
func getRecords() {
	request, err := http.NewRequest("GET", BASE_API_URL, nil)
	request.Header.Add("authorization", "Bearer " + cfg.Token)
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
					ZoneId:  zone.Id,
					Records: nil,
				})
				v4mutex := sync.Mutex{}
				v6mutex := sync.Mutex{}
				i, zone := i, zone
				go func() {
					defer wg.Done()
					request, err := http.NewRequest("GET", strings.Join([]string{BASE_API_URL, zone.Id, "/dns_records/"}, ""), nil)
					request.Header.Add("authorization", "Bearer " + cfg.Token)
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

func patchRecord(zone zoneAndRecords, record DnsRecord, ip string, wg *sync.WaitGroup) {
	defer wg.Done()
	request, err := http.NewRequest("PATCH", strings.Join([]string{BASE_API_URL, zone.ZoneId, "/dns_records/", record.Id}, ""),
		strings.NewReader("{\"content\":\""+ip+"\"}"))
	request.Header.Add("authorization", "Bearer " + cfg.Token)
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
		log.Print(apiResponse.Errors)
	}
}
