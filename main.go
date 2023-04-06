package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks"
	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks/geoip"
	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks/minpower"
)

// checks.NormalizedResponse, passFail, error
func handleRequest(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var formSubmission checks.FormSubmission
	err := json.Unmarshal([]byte(request.Body), &formSubmission)
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to deserialize request body: %v", err)
	}

	ctx := context.Background()

	// check miner power before geoip
	pass, err := checkMinerPower(ctx, formSubmission.MinerID)
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	if !pass {
		return events.APIGatewayProxyResponse{}, errors.New("miner power too low")
	}

	checker := geoip.GeoIPCheck{}

	location, err := checker.DoCheck(ctx, geoip.MinerData{
		MinerID:     formSubmission.MinerID,
		City:        formSubmission.City,
		CountryCode: formSubmission.Country,
	})
	if err != nil {
		log.Fatalln(err)
		return events.APIGatewayProxyResponse{}, err
	}

	contactInfoMap := map[string]string{
		"contact_name":  formSubmission.Name,
		"slack_id":      formSubmission.Slack,
		"contact_email": formSubmission.Email,
	}

	contactInfoJSON, err := json.Marshal(contactInfoMap)
	if err != nil {
		log.Fatalln(err)
		contactInfoJSON = []byte("{}") // TODO should fail here or continue?
	}

	// remove the f0 prefix from miner id to store as int in postgres
	minerIDInt, err := strconv.Atoi(formSubmission.MinerID[2:])
	if err != nil {
		log.Fatalln(err)
	}

	result := checks.NormalizedResponse{
		FormSubmission: formSubmission,
		NormalizedMiner: checks.NormalizedMiner{
			SPID:          minerIDInt,
			LocCity:       location.LocCity,
			LocCountry:    location.LocCountry,
			LocContinent:  location.LocContinent,
			Validated:     true,
			SPContactInfo: string(contactInfoJSON), // TODO: we need to seperate sp contact info
		},
		NormalizedOrg: checks.NormalizedOrg{
			SPOrgID:        "", // TODO: need to get appropriate OrgID or create one
			SPOrganization: formSubmission.SPName,
			OrgContactInfo: string(contactInfoJSON), // TODO: we need to seperate sp contact info
		},
	}

	jsonResponse, err := json.Marshal(result)
	if err != nil {
		return events.APIGatewayProxyResponse{}, fmt.Errorf("failed to serialize response: %v", err)
	}

	apiResponse := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(jsonResponse),
	}

	return apiResponse, nil
}

func checkMinerPower(ctx context.Context, minerID string) (bool, error) {
	// 10TiB = 10 * 1024^4 = 10995116277760 min power requirement
	min, ok := new(big.Int).SetString("10995116277760", 10)

	if !ok {
		return false, errors.New("failed to parse big int")
	}

	pass, err := minpower.MinQualityPowerOk(ctx, minerID, min)
	if err != nil {
		log.Fatalln(err)
		return false, err
	}

	if !pass {
		return false, errors.New("miner power too low")
	}

	return true, nil
}

func main() {
	lambda.Start(handleRequest)
}
