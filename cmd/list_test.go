package cmd

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestListingIDDecodesAsInteger(t *testing.T) {
	var response struct {
		Results []struct {
			ListingID int64 `json:"listing_id"`
		} `json:"results"`
	}
	if err := json.Unmarshal([]byte(`{"results":[{"listing_id":4546598563}]}`), &response); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%d", response.Results[0].ListingID); got != "4546598563" {
		t.Fatalf("listing ID = %q", got)
	}
}
