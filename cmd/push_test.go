package cmd

import "testing"

func TestListingFormFiltersReadOnlyFields(t *testing.T) {
	f := listingForm(map[string]any{"listing_id": 10, "title": "New title", "tags": []any{"one", "two"}})
	if f.Get("listing_id") != "" {
		t.Fatal("read-only listing_id was included")
	}
	if f.Get("title") != "New title" || f.Get("tags") != "one,two" {
		t.Fatalf("unexpected form: %v", f)
	}
}

func TestValidateCreate(t *testing.T) {
	if validateCreate(0, listingForm(map[string]any{"title": "x"})) == nil {
		t.Fatal("expected missing fields error")
	}
}

func TestListingFormConvertsMoney(t *testing.T) {
	f := listingForm(map[string]any{"price": map[string]any{"amount": float64(1299), "divisor": float64(100), "currency_code": "USD"}})
	if f.Get("price") != "12.99" {
		t.Fatalf("price = %q", f.Get("price"))
	}
}
