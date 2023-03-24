package main

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

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks"
)

// miner_id, city, country_code

func handleRequest(ctx context.Context, formResponse checks.FormSubmission) (checks.NormalizedResponse, error) {
	checkCtx := context.Background()

	var checks []checks.Check
	result := checks.NormalizedResponse{}

	err := getLocationData(ctx)
	if err != nil {
		log.Fatalln(err)
	}

	for _, c := range checks.RegisteredChecks {
		c.DoCheck(ctx, checkCtx, formResponse)
	}

	for _, err := range checkCtx.Errors {
		log.Fatalln(err)
		return nil, checkCtx.Errors
	}

	// json, err := testrig.RunChecksForFormResponses(context.Background(), formResponse, false)
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	return result, nil
}

func getLocationData(ctx context.Context) error {
	var tempDir string
	downloadsDir := "downloads"
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

	for _, dataUrl := range urls {
		u, err := url.Parse(dataUrl)
		if err != nil {
			return err
		}
		base := path.Base(u.Path)
		dest := path.Join(tempDir, base)

		if _, err := os.Stat(dest); errors.Is(err, os.ErrNotExist) {
			log.Printf("Downloading %s ...\n", base)
			resp, err := http.Get(dataUrl)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			out, err := os.Create(dest)
			if err != nil {
				return err
			}
			defer out.Close()
			io.Copy(out, resp.Body)
		}
	}
	return nil
}

func main() {
	lambda.Start(handleRequest)
}
