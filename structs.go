package main

import (
	"time"
)

type Config struct {
	Token    string        `fig:"token" validate:"required"`
	Domains  map[string]struct{
		V4Records	map[string]interface{} `fig:"v4-records"`
		V6Records	map[string]interface{} `fig:"v6-records"`
	}     `fig:"zones" validate:"required"`
	Timeout  time.Duration `fig:"timeout" default:"60s"`
}


type apiZones struct {
	Result []struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"result"`
	Success bool `json:"success"`
}

type apiRecords struct {
	Result  []DnsRecord `json:"result"`
	Success bool     `json:"success"`
	Errors  []apiError `json:"errors"`
}

type apiFeedback struct {
	Success bool     `json:"success"`
	Errors  []apiError `json:"errors"`
}

type apiError struct{
	Code	int `json:"code"`
	Message string `json:"message"`
}

type DnsRecord struct {
	Id   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type zoneAndRecords struct {
	ZoneId  string
	Records []DnsRecord
}
