package geoip

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"

	"github.com/pkg/errors"
)

type IPInfoResolver struct {
	Continents map[string]string
}

type IPInfoResponse struct {
	IP       string `json:"ip"`
	Hostname string `json:"hostname"`
	City     string `json:"city"`
	Region   string `json:"region"`
	Country  string `json:"country"`
	Location string `json:"loc"`
	Org      string `json:"org"`
	Postal   string `json:"postal"`
	Timezone string `json:"timezone"`
	ASN      struct {
		ASN    string `json:"asn"`
		Name   string `json:"name"`
		Domain string `json:"domain"`
		Route  string `json:"route"`
		Type   string `json:"type"`
	} `json:"asn"`
}

func NewIPInfoResolver() (*IPInfoResolver, error) {
	payload := make(map[string]string)

	if err := json.Unmarshal(CountryToContinentJSON, &payload); err != nil {
		return &IPInfoResolver{}, errors.Wrap(err, "ipinfo: failed to unmarshal continents")
	}

	return &IPInfoResolver{
		Continents: payload,
	}, nil
}

func (i *IPInfoResolver) ResolveIP(ctx context.Context, ip net.IP) (map[string]IPInfoResponse, error) {
	url := fmt.Sprintf("https://ipinfo.io/%s?token=%s", ip, os.Getenv("IPINFO_TOKEN"))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)

	if err != nil {
		return map[string]IPInfoResponse{}, errors.Wrap(err, "failed to create http request")
	}

	req.Header.Set("Accept", "application/json")
	client := &http.Client{}

	resp, err := client.Do(req)

	if err != nil {
		return map[string]IPInfoResponse{}, errors.Wrap(err, "failed to resolve IP")
	}

	defer resp.Body.Close()

	payload := make(map[string]IPInfoResponse)

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		return map[string]IPInfoResponse{}, errors.Wrap(err, "failed to read response body")
	}

	if err := json.Unmarshal(body, &payload); err != nil {
		return map[string]IPInfoResponse{}, errors.Wrap(err, "ipinfo: failed to unmarshal response")
	}

	return payload, nil
}

func (i *IPInfoResolver) ResolveIPStr(ctx context.Context, ip string) (string, error) {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return "", errors.Errorf("failed to parse IP address %s", ip)
	}

	result, err := i.ResolveIP(ctx, parsed)

	if err != nil {
		return "", errors.Wrap(err, "failed to resolve IP")
	}

	return result[ip].Country, nil
}
