package testrig

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"reflect"
	"strings"
)

type Miner struct {
	MinerID     string
	City        string
	CountryCode string
}

type MinerCheckResult struct {
	Miner   Miner
	Success bool

	OutputLines    []TestOutput
	ExtraArtifacts interface{}
}

type ResponseResult struct {
	ResponseFields    GoogleFormResponse
	MinerCheckResults MinerCheckResult
}

// "responseId": "ACYDBNg8yyGgwk051fZNDE5qZHAzZ_5YEfXpKl3XXZunSjsFZN8h2tSQghrDj2w-PK-QbB0",
// "timestamp": "2022-07-25T08:40:17.905455Z",
// "0_your_name": "test_name",
// "0_storage_provider_operator_name": "test_sp_name",
// "0_your_handle_on_filecoin_io_slack": "test_slack",
// "0_your_email": "test@example.com",
// "1_minerid": "f0478563",
// "1_city": "hangzhou",
// "1_country": "CN"
type GoogleFormResponse struct {
	Name    string `json:"your_name"`
	SPName  string `json:"storage_provider_operator_name"`
	Slack   string `json:"your_handle_on_filecoin_io_slack"`
	Email   string `json:"your_email"`
	MinerID string `json:"minerid"`
	City    string `json:"city"`
	Country string `json:"country"`
}

func RunChecksForFormResponses(ctx context.Context, formReponse GoogleFormResponse, forceEpoch bool) (string, error) {
	var responseResult ResponseResult

	// Download location data
	err := download_location_data(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed downloading location data: %w", err)
	}

	// Get the type and value of the struct
	t := reflect.TypeOf(formReponse)
	v := reflect.ValueOf(formReponse)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		fmt.Printf("%s: %v\n", field.Name, value.Interface())
	}

	miner := Miner{formReponse.MinerID, formReponse.City, formReponse.Country}
	log.Printf("Miner: %s - %s, %s\n", miner.MinerID, miner.City, miner.CountryCode)
	success, testOutput, extra, err := test_miner(ctx, miner, forceEpoch)

	log.Printf("Result: %v\n", success)
	if err != nil {
		log.Printf("Error: %v\n", err)
	}
	minerCheck := MinerCheckResult{miner, success, testOutput, extra}

	responseResult = ResponseResult{formReponse, minerCheck}

	jsonData, err := json.MarshalIndent(responseResult, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON marshal error: %w", err)
	}
	return string(jsonData), nil
}

func download_location_data(ctx context.Context) error {
	var tempDir string
	downloadsDir := "downloads"
	if _, err := os.Stat(fmt.Sprintf("/tmp/%s", downloadsDir)); errors.Is(err, os.ErrNotExist) {
		tempDir, err = ioutil.TempDir("/tmp", downloadsDir)
		if err != nil {
			log.Fatal("Failed to create temp dir:", err)
		}
	}

	defer func() {
		if _, err := os.Stat(fmt.Sprintf("/tmp/%s", downloadsDir)); errors.Is(err, os.ErrExist) {
			err := os.RemoveAll(tempDir)
			if err != nil {
				log.Fatal("Failed to remove temp dir:", err)
			}
		}
	}()

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

type TestOutput struct {
	Time    string
	Action  *string `json:",omitempty"`
	Package *string `json:",omitempty"`
	Test    *string `json:",omitempty"`
	Output  *string `json:",omitempty"`
}

func test_miner(ctx context.Context, miner Miner, forceEpoch bool) (bool,
	[]TestOutput, interface{}, error) {
	var outputLines []TestOutput
	var extra interface{} = nil

	extraArtifacts, err := os.CreateTemp("",
		fmt.Sprintf("extra-artifacts-%s-*.json", miner.MinerID))
	if err != nil {
		return false, outputLines, extra, err
	}
	err = extraArtifacts.Close()
	if err != nil {
		return false, outputLines, extra, err
	}
	defer os.Remove(extraArtifacts.Name())

	cmd := exec.CommandContext(
		ctx,
		"go",
		"./checks/minpower",
		"./checks/geoip",
		"-json",
	)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MINER_ID=%s", miner.MinerID),
		fmt.Sprintf("CITY=%s", miner.City),
		fmt.Sprintf("COUNTRY_CODE=%s", miner.CountryCode),
		"MULTIADDRS_IPS=../../downloads/multiaddrs-ips-latest.json",
		"IPS_GEOLITE2=../../downloads/ips-geolite2-latest.json",
		"IPS_BAIDU=../../downloads/ips-baidu-latest.json",
		"EXTRA_ARTIFACTS="+extraArtifacts.Name(),
	)
	// fmt.Println("EXTRA_ARTIFACTS", extraArtifacts.Name())
	if forceEpoch {
		cmd.Env = append(cmd.Env, "EPOCH=205500") // To match JSON files for testing
	}
	out, err := cmd.Output()
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		log.Println(line)
		outputLine := TestOutput{}
		json.Unmarshal([]byte(line), &outputLine)
		if outputLine.Time != "" {
			outputLines = append(outputLines, outputLine)
		}
	}
	extraData, err2 := ioutil.ReadFile(extraArtifacts.Name())
	err3 := json.Unmarshal(extraData, &extra)
	if err != nil || err2 != nil || err3 != nil {
		return false, outputLines, extra, err
	}
	return true, outputLines, extra, nil
}
