package geoip

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/jftuga/geodist"
	"github.com/savaki/geoip2"
	"googlemaps.github.io/maps"
)

const MAX_DISTANCE = 600
const downloadsDir = "downloads"

type GeoData struct {
	MultiaddrsIPs []MultiaddrsIPsRecord
	Ipinfo        *IPInfoResolver
	IPsGeolite2   map[string]IPsGeolite2Record
	IPsBaidu      map[string]IPsBaiduRecord
	IPsGeoIP2     map[string]geoip2.Response
}

func LoadGeoData() (*GeoData, error) {
	results, err := getLocationData()
	if err != nil {
		return nil, err
	}

	multiaddrsIPs, err := LoadMultiAddrsIPs(results[0])
	if err != nil {
		return nil, err
	}

	ipinfo, err := NewIPInfoResolver()
	if err != nil {
		return nil, err
	}

	ipsGeolite2, err := LoadIPsGeolite2(results[1])
	if err != nil {
		return nil, err
	}

	ipsBaidu, err := LoadIPsBaidu(results[2])
	if err != nil {
		return nil, err
	}

	return &GeoData{
		multiaddrsIPs,
		ipinfo,
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

	ipinfo, err := NewIPInfoResolver()
	if err != nil {
		return nil, err
	}

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
		ipinfo,
		ipsGeoLite2,
		ipsBaidu,
		ipsGeoIP2,
	}, nil
}

type FinalGeoData struct {
	GeoData           *GeoData
	GeocodeLocations  []geodist.Coord
	GeoDataAddresses  []Address
	GoogleGeocodeData []maps.GeocodingResult
}

func findMatchGeoLite2(g *GeoData, miner MinerData, locations []geodist.Coord) bool {
	var match_found bool = false
	for ip, geolite2 := range g.IPsGeolite2 {
		// Match country
		if geolite2.Country != miner.CountryCode {
			log.Printf("No Geolite2 country match for %s (%s != GeoLite2:%s), IP: %s\n",
				miner.MinerID, miner.CountryCode, geolite2.Country, ip)
			continue
		}
		log.Printf("Matching Geolite2 country for %s (%s) found, IP: %s\n",
			miner.MinerID, miner.CountryCode, ip)

		// Try to match city
		if geolite2.City == miner.City {
			log.Printf("Match found! %s matches Geolite2 city name (%s), IP: %s\n",
				miner.MinerID, miner.City, ip)
			match_found = true
			continue
		}
		log.Printf("No Geolite2 city match for %s (%s != GeoLite2:%s), IP: %s\n",
			miner.MinerID, miner.City, geolite2.City, ip)

		// Try to match based on Lat/Lng
		l := geolite2.Geolite2["location"].(map[string]interface{})
		geolite2Location := geodist.Coord{
			Lat: l["latitude"].(float64),
			Lon: l["longitude"].(float64),
		}
		log.Printf("Geolite2 Lat/Lng: %v for IP %s\n", geolite2Location, ip)

		// Distance based matching
		for i, location := range locations {
			log.Printf("Geocoded via Google %s, %s #%d Lat/Long %v", miner.City,
				miner.CountryCode, i+1, location)
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

func findMatchGeoIP2(g *GeoData, miner MinerData, locations []geodist.Coord) bool {
	provisional_match := false
	match_found := false
GEOIP2_LOOP:
	for ip, geoip2 := range g.IPsGeoIP2 {
		// Match country
		if geoip2.Country.IsoCode != miner.CountryCode {
			log.Printf("No GeoIP2 country match for %s (%s != GeoIP2:%s), IP: %s\n",
				miner.MinerID, miner.CountryCode, geoip2.Country.IsoCode, ip)
			continue
		}
		log.Printf("Matching GeoIP2 country for %s (%s) found, IP: %s\n",
			miner.MinerID, miner.CountryCode, ip)

		// Try to match city
		for _, cityName := range geoip2.City.Names {
			if cityName == miner.City {
				log.Printf("Match found! %s matches GeoIP2 city name (%s), IP: %s\n",
					miner.MinerID, miner.City, ip)
				match_found = true
				continue GEOIP2_LOOP
			}
		}
		log.Printf("No GeoIP2 city match for %s (%s != GeoIP2:%s), IP: %s\n",
			miner.MinerID, miner.City, geoip2.City.Names["en"], ip)
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
			log.Printf("Geocoded via Google %s, %s #%d Lat/Long %v", miner.City,
				miner.CountryCode, i+1, location)
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
			miner.MinerID)
		match_found = true
	}
	return match_found
}

func findMatchBaidu(g *GeoData, miner MinerData, locations []geodist.Coord) bool {
	match_found := false
	for ip, baidu := range g.IPsBaidu {
		// Try to match city
		if baidu.City == miner.City {
			log.Printf("Match found! %s matches city name (%s), IP: %s\n",
				miner.MinerID, miner.City, ip)
			match_found = true
			continue
		}
		log.Printf("No city match for %s (%s != Baidu:%s), IP: %s\n",
			miner.MinerID, miner.City, baidu.City, ip)

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
			log.Printf("Geocoded via Google %s, %s #%d Lat/Long %v", miner.City,
				miner.CountryCode, i+1, location)
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
func GeoMatchExists(
	ctx context.Context,
	geodata *GeoData,
	geocodeClient *maps.Client,
	currentEpoch int64,
	miner MinerData,
) (bool, FinalGeoData, error) {
	// Quick fixes for bad input data
	if miner.CountryCode == "United States" || miner.CountryCode == "San Jose, CA" {
		miner.CountryCode = "US"
	}
	if miner.CountryCode == "Canada" {
		miner.CountryCode = "CA"
	}
	miner.CountryCode = strings.ToUpper(miner.CountryCode)

	log.Printf("Searching for geo matches for %s (%s, %s)", miner.MinerID, miner.City, miner.CountryCode)
	g, err := geodata.filterByMinerID(ctx, miner.MinerID, currentEpoch)
	if err != nil {
		return false, FinalGeoData{}, err
	}

	data := FinalGeoData{GeoData: g}
	if err != nil {
		return false, data, err
	}

	if len(g.MultiaddrsIPs) == 0 {
		log.Printf("No Multiaddrs/IPs found for %s\n", miner.MinerID)
		return false, data, nil
	}

	locations, addresses, googleResponse, err := geocodeAddress(ctx, geocodeClient, fmt.Sprintf("%s, %s", miner.City, miner.CountryCode))
	if err != nil {
		log.Fatalf("Geocode error: %s", err)
	}
	data.GeocodeLocations = locations
	data.GeoDataAddresses = addresses
	data.GoogleGeocodeData = googleResponse

	match_found := false

	// First, try with Baidu data
	// if miner.CountryCode == "CN" {
	// 	match_found = findMatchBaidu(g, miner, locations)
	// }

	// // Next, try with Geolite2 data
	// if !match_found {
	// 	match_found = findMatchGeoLite2(g, miner, locations)
	// }

	// // last, try with GeoIP2 API data
	// if !match_found {
	// 	match_found = findMatchGeoIP2(g, miner, locations)
	// }

	if !match_found {
		log.Println("No match found.")
	}
	return match_found, data, nil
}

// returns a mapping of the tmp paths to the geodata downloads
func getLocationData() ([3]string, error) {
	var tempDir string
	if _, err := os.Stat(fmt.Sprintf("/tmp/%s", downloadsDir)); errors.Is(err, os.ErrNotExist) {
		tempDir, err = ioutil.TempDir("/tmp", downloadsDir)
		if err != nil {
			log.Fatal("Failed to create temp dir:", err)
		}
	}

	// not necessary as lambda removes all tmp files
	// TODO maybe make this more persistent cron so lambdas can share the latest data (cache)
	// defer func() {
	// 	if _, err := os.Stat(fmt.Sprintf("/tmp/%s", downloadsDir)); errors.Is(err, os.ErrExist) {
	// 		err := os.RemoveAll(tempDir)
	// 		if err != nil {
	// 			log.Fatal("Failed to remove temp dir:", err)
	// 		}
	// 	}
	// }()

	urls := []string{
		"https://multiaddrs-ips.feeds.provider.quest/multiaddrs-ips-latest.json",
		"https://geoip.feeds.provider.quest/ips-geolite2-latest.json",
		"https://geoip.feeds.provider.quest/ips-baidu-latest.json",
	}

	result := [3]string{}

	for i, dataUrl := range urls {
		u, err := url.Parse(dataUrl)
		if err != nil {
			return [3]string{}, err
		}
		base := path.Base(u.Path)
		dest := path.Join(tempDir, base)

		if _, err := os.Stat(dest); errors.Is(err, os.ErrNotExist) {
			log.Printf("Downloading %s ...\n", base)
			resp, err := http.Get(dataUrl)
			if err != nil {
				return [3]string{}, err
			}
			defer resp.Body.Close()
			out, err := os.Create(dest)
			if err != nil {
				return [3]string{}, err
			}
			defer out.Close()
			io.Copy(out, resp.Body)
		}

		result[i] = dest
	}

	return result, nil
}
