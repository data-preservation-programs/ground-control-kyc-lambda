package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math/big"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks"
	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks/geoip"
	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks/minpower"
)

type passFail bool

func handleRequest(ctx context.Context, formSubmission checks.FormSubmission) (checks.NormalizedResponse, passFail, error) {
	// check miner power before geoip
	pass, err := checkMinerPower(ctx, formSubmission.MinerID)
	if err != nil {
		return checks.NormalizedResponse{}, false, err
	}

	if !pass {
		return checks.NormalizedResponse{}, false, errors.New("miner power too low")
	}

	checker := geoip.GeoIPCheck{}

	location, err := checker.DoCheck(ctx, geoip.MinerData{
		MinerID:     formSubmission.MinerID,
		City:        formSubmission.City,
		CountryCode: formSubmission.Country,
	})
	if err != nil {
		log.Fatalln(err)
		return checks.NormalizedResponse{}, false, err
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
			SPOrgID:        "",
			SPOrganization: formSubmission.SPName,
			OrgContactInfo: string(contactInfoJSON), // TODO: we need to seperate sp contact info
		},
	}

	return result, true, nil
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
