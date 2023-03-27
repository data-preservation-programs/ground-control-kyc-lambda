package geoip

import (
	"encoding/json"
	"os"
)

type IPsGeolite2Report struct {
	Date *string                      `json:"date"`
	IPs  map[string]IPsGeolite2Record `json:"ipsGeolite2"`
}

type IPsGeolite2Record struct {
	Epoch     uint64  `json:"epoch"`
	Timestamp string  `json:"timestamp"`
	Continent string  `json:"continent"`
	Country   string  `json:"country"`
	Subdiv1   string  `json:"subdiv1"`
	City      string  `json:"city"`
	Long      float32 `json:"long"`
	Lat       float32 `json:"lat"`
	Geolite2  Geolite2Detail
}

type Geolite2Detail map[string]interface{}

func LoadIPsGeolite2(filepath string) (map[string]IPsGeolite2Record, error) {
	if filepath == "" {
		filepath = "testdata/ips-geolite2-latest.json"
	}
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var report IPsGeolite2Report
	json.Unmarshal(bytes, &report)
	return report.IPs, nil
}
