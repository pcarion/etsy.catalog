package catalog

import (
	"os"
	"strings"
	"testing"
)

func TestSaveFormatsEtsyValuesForHumans(t *testing.T) {
	root := t.TempDir()
	doc := &Document{
		CatalogID: "4546598563",
		Listing: map[string]any{
			"listing_id":        float64(4546598563),
			"description":       "First paragraph.  \n\nWHAT&#39;S INCLUDED",
			"creation_tsz":      float64(1785428193),
			"created_timestamp": float64(1785381989),
			"score":             4.5,
		},
	}
	if err := Save(root, doc.CatalogID, doc); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(Dir(root, doc.CatalogID) + "/listing.yaml")
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{
		"listing_id: 4546598563",
		"description: |-\n        First paragraph.\n\n        WHAT'S INCLUDED",
		"creation_tsz: \"2026-07-30T16:16:33Z\"",
		"created_timestamp: \"2026-07-30T03:26:29Z\"",
		"score: 4.5",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("generated YAML does not contain %q:\n%s", want, got)
		}
	}
}
