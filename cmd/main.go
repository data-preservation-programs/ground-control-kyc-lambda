package main

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks"
)

// miner_id, city, country_code

var RegisteredChecks []checks.Check

func handleRequest(ctx context.Context, formSubmission checks.FormSubmission) (checks.NormalizedResponse, error) {
	checkCtx := context.Background()

	for _, c := range RegisteredChecks {
		c.DoCheck(ctx, checkCtx, formSubmission)
	}

	for _, err := range checkCtx.Errors {
		log.Fatalln(err)
		return nil, checkCtx.Errors
	}

	contactInfoMap := map[string]string{
		"your_name":                        formSubmission.Name,
		"your_handle_on_filecoin_io_slack": formSubmission.Slack,
		"your_email":                       formSubmission.Email,
	}

	contactInfoJSON, err := json.Marshal(contactInfoMap)
	if err != nil {
		log.Fatalln(err)
		contactInfoJSON = []byte("{}") // TODO should fail here or continue?
	}

	// remove the f0 prefix from miner id to store as int in postgres
	minerIDInt, err := strconv.Atoi(formSubmission.MinerID[2:])

	result := checks.NormalizedResponse{
		FormSubmission: formSubmission,
		NormalizedMiner: checks.NormalizedMiner{
			SPID:          minerIDInt,
			LocCity:       "",
			LocCountry:    "",
			LocContinent:  "",
			Validated:     true,
			SPContactInfo: string(contactInfoJSON), // TODO: we need to seperate sp contact info
		},
		NormalizedOrg: checks.NormalizedOrg{
			SPOrgID:        "",
			SPOrganization: formSubmission.SPName,
			OrgContactInfo: string(contactInfoJSON), // TODO: we need to seperate sp contact info
		},
	}

	return result, nil
}

func main() {
	lambda.Start(handleRequest)
}
