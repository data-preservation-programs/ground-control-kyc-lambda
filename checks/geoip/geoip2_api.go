package geoip

import (
	"context"
	"log"
	"os"

	"github.com/savaki/geoip2"
)

func getGeoIP2(ctx context.Context, ip string) (geoip2.Response, error) {
	userid := os.Getenv("MAXMIND_USER_ID")
	key := os.Getenv("MAXMIND_LICENSE_KEY")
	if userid == "skip" || key == "skip" {
		log.Println("Warning: Skipping Maxmind GeoIP2 API lookups")
		return geoip2.Response{}, nil
	}
	api := geoip2.New(userid, key)
	return api.Insights(ctx, ip)
}
