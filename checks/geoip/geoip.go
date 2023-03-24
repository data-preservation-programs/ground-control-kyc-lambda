package geoip

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks"
)

type GeoIPCheck struct{}

func init() {
	checks.Register(GeoIPCheck{})
}

type MinerData struct {
	MinerID     string `json:"miner_id"`
	City        string `json:"city"`
	CountryCode string `json:"country_code"`
}

func (*GeoIPCheck) DoCheck(ctx context.Context, checkContext checksContext) {
	var err error
	currentEpoch, err := strconv.ParseInt(os.Getenv("EPOCH"), 10, 64)
	if currentEpoch == 0 || err != nil {
		currentEpoch, err = GetCurrentEpoch(context.Background())
		if err != nil {
			log.Fatalf("Error getting current epoch: %v\n", err)
		}
	}

	c := MinerData{minerID, city, countryCode}

	geodata, err := LoadGeoData()

	geocodeClient, err := GetGeocodeClient()

	ok, extra, err := GeoMatchExists(context.Background(), geodata, geocodeClient, currentEpoch, c.minerID, c.city, c.countryCode)

	extraJson, err := json.MarshalIndent(extra, "", "  ")
	ioutil.WriteFile(extraArtifacts, extraJson, 0644)
}
