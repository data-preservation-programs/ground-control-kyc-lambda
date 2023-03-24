package geoip

import (
	"encoding/json"
	"os"
)

type IPsBaiduReport struct {
	Date *string                   `json:"date"`
	IPs  map[string]IPsBaiduRecord `json:"ipsBaidu"`
}

type IPsBaiduRecord struct {
	Epoch     uint64      `json:"epoch"`
	Timestamp string      `json:"timestamp"`
	City      string      `json:"city"`
	Long      float32     `json:"long"`
	Lat       float32     `json:"lat"`
	Baidu     BaiduDetail `json:"baidu"`
}

type BaiduDetail map[string]interface{}

func LoadIPsBaidu(filepath string) (map[string]IPsBaiduRecord, error) {
	if filepath == "" {
		filepath = "testdata/ips-baidu-latest.json"
	}
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var report IPsBaiduReport
	json.Unmarshal(bytes, &report)
	return report.IPs, nil
}
