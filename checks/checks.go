package checks

import "context"

// "responseId": "ACYDBNg8yyGgwk051fZNDE5qZHAzZ_5YEfXpKl3XXZunSjsFZN8h2tSQghrDj2w-PK-QbB0",
// "timestamp": "2022-07-25T08:40:17.905455Z",
// "0_your_name": "test_name",
// "0_storage_provider_operator_name": "test_sp_name",
// "0_your_handle_on_filecoin_io_slack": "test_slack",
// "0_your_email": "test@example.com",
// "1_minerid": "f0478563",
// "1_city": "hangzhou",
// "1_country": "CN"
type FormSubmission struct {
	Name    string `json:"your_name"`
	SPName  string `json:"storage_provider_operator_name"`
	Slack   string `json:"your_handle_on_filecoin_io_slack"`
	Email   string `json:"your_email"`
	MinerID string `json:"minerid"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type NormalizedLocation struct {
	LocCity      string `json:"loc_city"`
	LocCountry   string `json:"loc_country"`
	LocContinent string `json:"loc_continent"`
}

type NormalizedMiner struct {
	SPID          int    `json:"sp_id"`
	LocCity       string `json:"loc_city"`
	LocCountry    string `json:"loc_country"`
	LocContinent  string `json:"loc_continent"`
	Validated     bool   `json:"validated"`
	SPContactInfo string `json:"contact_info"`
}

type NormalizedOrg struct {
	SPOrgID        string `json:"sp_org_id"`
	SPOrganization string `json:"sp_organization"`
	OrgContactInfo string `json:"org_contact_info"`
}

type NormalizedResponse struct {
	FormSubmission  FormSubmission
	NormalizedMiner NormalizedMiner
	NormalizedOrg   NormalizedOrg
}

type Check interface {
	DoCheck(ctx context.Context, checkctx context.Context, args ...string) (NormalizedResponse, error)
}

// could be a map[string]Check
var RegisteredChecks []Check

func Register(c Check) {
	RegisteredChecks = append(RegisteredChecks, c)
}

// sort by priority
// sort.Slice(RegisteredChecks, func(...))
// GetPriority() int
