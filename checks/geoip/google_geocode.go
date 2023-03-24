package geoip

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jftuga/geodist"
	"googlemaps.github.io/maps"
)

func GetGeocodeClient() (*maps.Client, error) {
	key := os.Getenv("GOOGLE_MAPS_API_KEY")
	if key == "" {
		log.Fatalf("Missing GOOGLE_MAPS_API_KEY")
	}
	if key == "skip" {
		log.Println("Warning: GOOGLE_MAPS_API_KEY set to 'skip'")
		return nil, nil
	}
	return maps.NewClient(maps.WithAPIKey(key))
}

type Address struct {
	StreetNumber string `json:"street_number"`
	Route        string `json:"route"`
	City         string `json:"city"`
	State        string `json:"state"`
	CityState    string `json:"city_state"`
	Country      string `json:"country"`
}

func getAddressComponents(components []maps.AddressComponent) Address {
	var address Address

	for _, c := range components {
		if contains(c.Types, "street_number") {
			address.StreetNumber = c.LongName
		} else if contains(c.Types, "route") {
			address.Route = c.LongName
		} else if contains(c.Types, "locality") {
			if address.Country == "United States" {
				address.CityState = fmt.Sprintf("%s, %s", c.LongName, address.State)
				if address.State != "" {
					address.CityState = fmt.Sprintf("%s, %s", address.CityState, address.State)
				}
			} else {
				address.City = c.LongName
			}
		} else if contains(c.Types, "administrative_area_level_1") {
			if address.Country == "United States" {
				address.State = fmt.Sprintf("%s, %s", c.ShortName, address.State)
				if address.City != "" {
					address.CityState = fmt.Sprintf("%s, %s", address.City, address.State)
				}
			} else {
				address.State = c.LongName
			}
		} else if contains(c.Types, "country") {
			address.Country = c.ShortName
		}
	}

	return address
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

func geocodeAddress(ctx context.Context, client *maps.Client, address string) ([]geodist.Coord, []Address, []maps.GeocodingResult, error) {
	if client == nil {
		return []geodist.Coord{}, []Address{}, []maps.GeocodingResult{}, nil
	}

	r := &maps.GeocodingRequest{
		Address: address,
	}
	resp, err := client.Geocode(ctx, r)
	if err != nil {
		return []geodist.Coord{}, []Address{}, resp, err
	}

	var locations []geodist.Coord
	var addresses []Address
	for _, r := range resp {
		location := geodist.Coord{
			Lat: r.Geometry.Location.Lat,
			Lon: r.Geometry.Location.Lng,
		}
		locations = append(locations, location)

		addr := getAddressComponents(r.AddressComponents)
		addresses = append(addresses, addr)
	}

	// TODO add address response
	return locations, addresses, resp, nil
}
