package geoip

import (
	"encoding/json"
	"os"
)

type MultiaddrsIPsReport struct {
	Date          *string
	MultiaddrsIPs []MultiaddrsIPsRecord
}

type MultiaddrsIPsRecord struct {
	Miner     string `json:"miner"`
	Maddr     string `json:"maddr"`
	PeerID    string `json:"peerId"`
	IP        string `json:"ip"`
	Epoch     uint   `json:"epoch"`
	Timestamp string `json:"timestamp"`
	DHT       bool   `json:"dht"`
	Chain     bool   `json:"chain"`
}

func LoadMultiAddrsIPs() ([]MultiaddrsIPsRecord, error) {
	file := os.Getenv("MULTIADDRS_IPS")
	if file == "" {
		file = "testdata/multiaddrs-ips-latest.json"
	}
	bytes, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var report MultiaddrsIPsReport
	json.Unmarshal(bytes, &report)
	return report.MultiaddrsIPs, nil
}
