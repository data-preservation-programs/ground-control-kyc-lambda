package geoip

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/jftuga/geodist"
	"github.com/savaki/geoip2"
	"googlemaps.github.io/maps"
)

const MAX_DISTANCE = 600

type GeoData struct {
	MultiaddrsIPs []MultiaddrsIPsRecord
	IPsGeolite2   map[string]IPsGeolite2Record
	IPsBaidu      map[string]IPsBaiduRecord
	IPsGeoIP2     map[string]geoip2.Response
}

func LoadGeoData() (*GeoData, error) {
	multiaddrsIPs, err := LoadMultiAddrsIPs()
	if err != nil {
		return nil, err
	}

	ipsGeolite2, err := LoadIPsGeolite2()
	if err != nil {
		return nil, err
	}

	ipsBaidu, err := LoadIPsBaidu()
	if err != nil {
		return nil, err
	}

	return &GeoData{
		multiaddrsIPs,
		ipsGeolite2,
		ipsBaidu,
		make(map[string]geoip2.Response),
	}, nil
}

func (g *GeoData) filterByMinerID(ctx context.Context, minerID string, currentEpoch int64) (*GeoData, error) {
	minEpoch := currentEpoch - 14*24*60*2 // 2 weeks
	multiaddrsIPs := []MultiaddrsIPsRecord{}
	ipsGeoLite2 := make(map[string]IPsGeolite2Record)
	ipsBaidu := make(map[string]IPsBaiduRecord)
	ipsGeoIP2 := make(map[string]geoip2.Response)
	for _, m := range g.MultiaddrsIPs {
		if m.Miner == minerID {
			if int64(m.Epoch) < minEpoch {
				log.Printf("IP address %s rejected, too old: %d < %d\n",
					m.IP, m.Epoch, minEpoch)
			} else {
				multiaddrsIPs = append(multiaddrsIPs, m)
				if r, ok := g.IPsGeolite2[m.IP]; ok {
					ipsGeoLite2[m.IP] = r
				}
				if r, ok := g.IPsBaidu[m.IP]; ok {
					ipsBaidu[m.IP] = r
				}
				r, err := getGeoIP2(ctx, m.IP)
				if err != nil {
					return &GeoData{}, err
				}
				ipsGeoIP2[m.IP] = r
			}
		}
	}

	return &GeoData{
		multiaddrsIPs,
		ipsGeoLite2,
		ipsBaidu,
		ipsGeoIP2,
	}, nil
}

type ExtraArtifacts struct {
	GeoData           *GeoData
	GeocodeLocations  []geodist.Coord
	GeoDataAddresses  []Address
	GoogleGeocodeData []maps.GeocodingResult
}

func findMatchGeoLite2(g *GeoData, minerID string, city string,
	countryCode string, locations []geodist.Coord) bool {
	var match_found bool = false
	for ip, geolite2 := range g.IPsGeolite2 {
		// Match country
		if geolite2.Country != countryCode {
			log.Printf("No Geolite2 country match for %s (%s != GeoLite2:%s), IP: %s\n",
				minerID, countryCode, geolite2.Country, ip)
			continue
		}
		log.Printf("Matching Geolite2 country for %s (%s) found, IP: %s\n",
			minerID, countryCode, ip)

		// Try to match city
		if geolite2.City == city {
			log.Printf("Match found! %s matches Geolite2 city name (%s), IP: %s\n",
				minerID, city, ip)
			match_found = true
			continue
		}
		log.Printf("No Geolite2 city match for %s (%s != GeoLite2:%s), IP: %s\n",
			minerID, city, geolite2.City, ip)

		// Try to match based on Lat/Lng
		l := geolite2.Geolite2["location"].(map[string]interface{})
		geolite2Location := geodist.Coord{
			Lat: l["latitude"].(float64),
			Lon: l["longitude"].(float64),
		}
		log.Printf("Geolite2 Lat/Lng: %v for IP %s\n", geolite2Location, ip)

		// Distance based matching
		for i, location := range locations {
			log.Printf("Geocoded via Google %s, %s #%d Lat/Long %v", city,
				countryCode, i+1, location)
			_, distance, err := geodist.VincentyDistance(location, geolite2Location)
			if err != nil {
				log.Println("Unable to compute Vincenty Distance.")
				continue
			} else {
				if distance <= MAX_DISTANCE {
					log.Printf("Match found! Distance %f km\n", distance)
					match_found = true
					continue
				}
				log.Printf("No match, distance %f km > %d km\n", distance, MAX_DISTANCE)
			}
		}
	}
	return match_found
}

func findMatchGeoIP2(g *GeoData, minerID string, city string,
	countryCode string, locations []geodist.Coord) bool {
	provisional_match := false
	match_found := false
GEOIP2_LOOP:
	for ip, geoip2 := range g.IPsGeoIP2 {
		// Match country
		if geoip2.Country.IsoCode != countryCode {
			log.Printf("No GeoIP2 country match for %s (%s != GeoIP2:%s), IP: %s\n",
				minerID, countryCode, geoip2.Country.IsoCode, ip)
			continue
		}
		log.Printf("Matching GeoIP2 country for %s (%s) found, IP: %s\n",
			minerID, countryCode, ip)

		// Try to match city
		for _, cityName := range geoip2.City.Names {
			if cityName == city {
				log.Printf("Match found! %s matches GeoIP2 city name (%s), IP: %s\n",
					minerID, city, ip)
				match_found = true
				continue GEOIP2_LOOP
			}
		}
		log.Printf("No GeoIP2 city match for %s (%s != GeoIP2:%s), IP: %s\n",
			minerID, city, geoip2.City.Names["en"], ip)
		if geoip2.City.Names["en"] == "" {
			provisional_match = true
		}

		// Try to match based on Lat/Lng
		geoip2Location := geodist.Coord{
			Lat: geoip2.Location.Latitude,
			Lon: geoip2.Location.Longitude,
		}
		log.Printf("GeoIP2 Lat/Lng: %v for IP %s\n", geoip2Location, ip)

		// Distance based matching
		for i, location := range locations {
			log.Printf("Geocoded via Google %s, %s #%d Lat/Long %v", city,
				countryCode, i+1, location)
			_, distance, err := geodist.VincentyDistance(location, geoip2Location)
			if err != nil {
				log.Println("Unable to compute Vincenty Distance.")
				continue
			} else {
				if distance <= MAX_DISTANCE {
					log.Printf("Match found! Distance %f km\n", distance)
					match_found = true
					continue
				}
				log.Printf("No match, distance %f km > %d km\n", distance, MAX_DISTANCE)
			}
		}
	}
	if provisional_match {
		log.Printf("Match found! %s had GeoIP2 entries that matched country, "+
			"all with no city data.\n",
			minerID)
		match_found = true
	}
	return match_found
}

func findMatchBaidu(g *GeoData, minerID string, city string,
	countryCode string, locations []geodist.Coord) bool {
	match_found := false
	for ip, baidu := range g.IPsBaidu {
		// Try to match city
		if baidu.City == city {
			log.Printf("Match found! %s matches city name (%s), IP: %s\n",
				minerID, city, ip)
			match_found = true
			continue
		}
		log.Printf("No city match for %s (%s != Baidu:%s), IP: %s\n",
			minerID, city, baidu.City, ip)

		baiduContent := baidu.Baidu["content"].(map[string]interface{})
		baiduPoint := baiduContent["point"].(map[string]interface{})
		lon, err := strconv.ParseFloat(baiduPoint["x"].(string), 64)
		if err != nil {
			log.Println("Error parsing baidu longitude (x)", err)
			continue
		}
		lat, err := strconv.ParseFloat(baiduPoint["y"].(string), 64)
		if err != nil {
			log.Println("Error parsing baidu latitude (y)", err)
			continue
		}
		baiduLocation := geodist.Coord{
			Lat: lat,
			Lon: lon,
		}
		log.Printf("Baidu Lat/Lng: %v for IP %s\n", baiduLocation, ip)
		// Distance based matching
		for i, location := range locations {
			log.Printf("Geocoded via Google %s, %s #%d Lat/Long %v", city,
				countryCode, i+1, location)
			_, distance, err := geodist.VincentyDistance(location, baiduLocation)
			if err != nil {
				log.Println("Unable to compute Vincenty Distance.")
				continue
			} else {
				if distance <= MAX_DISTANCE {
					log.Printf("Match found! Distance %f km\n", distance)
					match_found = true
					continue
				}
				log.Printf("No match, distance %f km > %d km\n", distance, MAX_DISTANCE)
			}
		}
	}
	return match_found
}

// GeoMatchExists checks if the miner has an IP address with a location close to the city/country
func GeoMatchExists(ctx context.Context, geodata *GeoData,
	geocodeClient *maps.Client, currentEpoch int64, minerID string, city string,
	countryCode string) (bool, ExtraArtifacts, error) {

	// Quick fixes for bad input data
	if countryCode == "United States" || countryCode == "San Jose, CA" {
		countryCode = "US"
	}
	if countryCode == "Canada" {
		countryCode = "CA"
	}
	countryCode = strings.ToUpper(countryCode)

	log.Printf("Searching for geo matches for %s (%s, %s)", minerID,
		city, countryCode)
	g, err := geodata.filterByMinerID(ctx, minerID, currentEpoch)
	extraArtifacts := ExtraArtifacts{GeoData: g}
	if err != nil {
		return false, extraArtifacts, err
	}

	if len(g.MultiaddrsIPs) == 0 {
		log.Printf("No Multiaddrs/IPs found for %s\n", minerID)
		return false, extraArtifacts, nil
	}

	locations, addresses, googleResponse, err := geocodeAddress(ctx, geocodeClient,
		fmt.Sprintf("%s, %s", city, countryCode))
	if err != nil {
		log.Fatalf("Geocode error: %s", err)
	}
	extraArtifacts.GeocodeLocations = locations
	extraArtifacts.GeoDataAddresses = addresses
	extraArtifacts.GoogleGeocodeData = googleResponse

	match_found := false

	// First, try with Baidu data
	if countryCode == "CN" {
		match_found = findMatchBaidu(g, minerID, city, countryCode, locations)
	}

	// Next, try with Geolite2 data
	if !match_found {
		match_found = findMatchGeoLite2(g, minerID, city, countryCode, locations)
	}

	// last, try with GeoIP2 API data
	if !match_found {
		match_found = findMatchGeoIP2(g, minerID, city, countryCode, locations)
	}

	if !match_found {
		log.Println("No match found.")
	}
	return match_found, extraArtifacts, nil
}
