package geoip

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strconv"

	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks"
)

type GeoIPCheck struct{}

// func init() {
// 	checks.Register(GeoIPCheck{})
// }

type MinerData struct {
	MinerID     string `json:"miner_id"`
	City        string `json:"city"`
	CountryCode string `json:"country_code"`
}

func (*GeoIPCheck) DoCheck(ctx context.Context, miner MinerData) (checks.NormalizedLocation, error) {
	var err error
	currentEpoch, err := strconv.ParseInt(os.Getenv("EPOCH"), 10, 64)
	if currentEpoch == 0 || err != nil {
		currentEpoch, err = GetCurrentEpoch(context.Background())
		if err != nil {
			log.Fatalf("Error getting current epoch: %v\n", err)
		}
	}

	continentCodesJSON, err := ioutil.ReadFile("./continents.json")
	if err != nil {
		fmt.Println(err)
	}

	var continentCodes map[string]string
	err = json.Unmarshal(continentCodesJSON, &continentCodes)
	if err != nil {
		fmt.Println(err)
	}

	geodata, err := LoadGeoData()
	if err != nil {
		return checks.NormalizedLocation{}, err
	}

	geocodeClient, err := GetGeocodeClient()
	if err != nil {
		return checks.NormalizedLocation{}, err
	}

	ok, data, err := GeoMatchExists(context.Background(), geodata, geocodeClient, currentEpoch, miner)
	if !ok {
		return checks.NormalizedLocation{}, err
	}

	continent, ok := continentCodes[data.GeoDataAddresses[0].Country]
	if !ok {
		continent := continentCodes[miner.CountryCode]
		if continent != "" {
			fmt.Printf("Continent %s found for country %s ", continent, miner.CountryCode)
		} else {
			fmt.Println("Continent not found")
			// bail?
		}
	}

	response := checks.NormalizedLocation{
		LocCity:      data.GeoDataAddresses[0].CityState,
		LocCountry:   data.GeoDataAddresses[0].Country,
		LocContinent: continent,
	}

	return response, nil
}
