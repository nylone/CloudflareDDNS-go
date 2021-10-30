package main

import (
	"time"
)

type config struct {
	Token    string        `fig:"token" validate:"required"`
	Timeout  time.Duration `fig:"timeout" default:"60s"`
	DoIPv4   bool          `fig:"do-ipv4"`
	DoIPv6   bool          `fig:"do-ipv6"`
	LogLevel string        `fig:"loglevel" default:"error"`
	Domains  map[string]struct {
		V4Records map[string]interface{} `fig:"v4-records"`
		V6Records map[string]interface{} `fig:"v6-records"`
	} `fig:"zones" validate:"required"`
}

type apiZones struct {
	Result []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"result"`
	Success bool `json:"success"`
}

type apiRecords struct {
	Result  []dnsRecord `json:"result"`
	Success bool        `json:"success"`
	Errors  []apiError  `json:"errors"`
}

type apiFeedback struct {
	Success bool       `json:"success"`
	Errors  []apiError `json:"errors"`
}

type apiError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type dnsRecord struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type zoneAndRecords struct {
	ZoneID  string
	Records []dnsRecord
}
