package service

import (
	"encoding/json"
	"testing"

	"github.com/ianzepp/gofolio-api/internal/calculator"
)

func TestMatchesFilterHappyPath(t *testing.T) {
	activity := calculator.Activity{
		AccountID:     "acct-1",
		AssetClass:    "EQUITY",
		DataSource:    "YAHOO",
		Symbol:        "AAPL",
		Tags:          []calculator.Tag{{ID: "growth"}},
		AssetSubClass: "STOCK",
	}
	filter := &PortfolioFilter{
		AccountIDs:   []string{"acct-1"},
		AssetClasses: []string{"EQUITY"},
		DataSource:   "YAHOO",
		Symbol:       "AAPL",
		Tags:         []string{"growth"},
	}
	if !matchesFilter(activity, filter) {
		t.Fatalf("expected activity to match filter")
	}
}

func TestMatchesFilterRejectsMismatchedTags(t *testing.T) {
	activity := calculator.Activity{
		AccountID:  "acct-1",
		AssetClass: "EQUITY",
		DataSource: "YAHOO",
		Symbol:     "AAPL",
		Tags:       []calculator.Tag{{ID: "growth"}},
	}
	filter := &PortfolioFilter{Tags: []string{"income"}}
	if matchesFilter(activity, filter) {
		t.Fatalf("expected filter mismatch when tag not present")
	}
}

func TestTagsToAnyHappyPath(t *testing.T) {
	tags := []calculator.Tag{{ID: "t1", Name: "Growth"}, {ID: "t2", Name: "Value"}}
	items := tagsToAny(tags)
	if len(items) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(items))
	}
	first := items[0].(map[string]string)
	if first["id"] != "t1" || first["name"] != "Growth" {
		t.Fatalf("unexpected map contents: %v", first)
	}
}

func TestTagsToAnyEmpty(t *testing.T) {
	items := tagsToAny(nil)
	if len(items) != 0 {
		t.Fatalf("expected empty slice, got %d", len(items))
	}
}

func TestEnsureJSONArrayHappyPath(t *testing.T) {
	input := json.RawMessage("[1]")
	if string(ensureJSONArray(input)) != "[1]" {
		t.Fatalf("expected original array to pass through")
	}
	if string(ensureJSONArray(nil)) != "[]" {
		t.Fatalf("expected fallback [] for nil input")
	}
}

func TestEnsureJSONObjectHappyPath(t *testing.T) {
	input := json.RawMessage("{\"k\":1}")
	if string(ensureJSONObject(input)) != "{\"k\":1}" {
		t.Fatalf("expected original object to pass through")
	}
	if string(ensureJSONObject(nil)) != "{}" {
		t.Fatalf("expected fallback {} for nil input")
	}
}
