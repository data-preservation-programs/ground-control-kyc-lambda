package main

import (
	"context"
	"log"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/data-preservation-programs/ground-control-kyc-lambda/checks"
)

// miner_id, city, country_code

func handleRequest(ctx context.Context, formResponse testrig.GoogleFormResponse) (string, error) {
	checkCtx := checks.Context{}

	var checks []checks.Check
	var result map[checks.Check]

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

func main() {
	lambda.Start(handleRequest)
}
