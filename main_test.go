package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type formSubmission struct {
	SlackID     string   `json:"slack_id"`
	Name        string   `json:"name"`
	SPIDs       []string `json:"sp_ids"`
	CompanyName string   `json:"company_name"`
	City        string   `json:"city"`
	Country     string   `json:"country"`
	StartDate   string   `json:"Start Date (UTC)"`
	SubmitDate  string   `json:"Submit Date (UTC)"`
	NetworkID   string   `json:"Network ID"`
	Tags        string   `json:"Tags"`
}

func LambdaTest(t *testing.T) {
	cases := []formSubmission{
		{
			SlackID:     "user_123",
			Name:        "John Doe",
			SPIDs:       []string{"f012345"},
			CompanyName: "ABC Inc.",
			City:        "New York",
			Country:     "USA",
			StartDate:   "2023-04-10 09:15:00",
			SubmitDate:  "2023-04-10 09:18:12",
			NetworkID:   "a43fj39",
			Tags:        "tag1, tag2, tag3",
		},
		{
			SlackID:     "user_456",
			Name:        "Jane Smith",
			SPIDs:       []string{"f067890"},
			CompanyName: "XYZ Corp.",
			City:        "Los Angeles",
			Country:     "USA",
			StartDate:   "2023-04-12 13:30:00",
			SubmitDate:  "2023-04-12 13:35:45",
			NetworkID:   "b49fng94",
			Tags:        "tag2, tag4",
		},
	}

	for _, c := range cases {
		// TODO test cases - dummy test for now
		assert.Equal(t, c.SlackID, "user_123")
	}
}
