package main

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestTemporalRuleExample(t *testing.T) {
	opts := createTemporalRuleModel()

	var buf bytes.Buffer
	if err := goat.Debug(&buf, opts...); err != nil {
		t.Fatalf("Debug failed: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	expectedWorldsData, err := os.ReadFile("expected_worlds.json")
	if err != nil {
		t.Fatalf("Failed to read expected worlds file: %v", err)
	}

	var expectedWorlds any
	if err := json.Unmarshal(expectedWorldsData, &expectedWorlds); err != nil {
		t.Fatalf("Failed to parse expected worlds JSON: %v", err)
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreMapEntries(func(k, v any) bool {
			key, ok := k.(string)
			return ok && key == "summary"
		}),
	}

	if diff := cmp.Diff(expectedWorlds, data["worlds"], cmpOpts...); diff != "" {
		t.Errorf("Worlds mismatch (-expected +actual):\n%s", diff)
	}

	temporal, ok := data["temporal_rules"].([]any)
	if !ok || len(temporal) != 1 {
		t.Fatalf("expected one temporal rule, got: %v", data["temporal_rules"])
	}
	rule, ok := temporal[0].(map[string]any)
	if !ok {
		t.Fatalf("malformed temporal rule data: %v", temporal[0])
	}
	if rule["holds"] != true {
		t.Fatalf("expected temporal rule to hold")
	}
}
