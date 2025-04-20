package main

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestMeetingRoomReservationWithExclusion(t *testing.T) {
	opts := createMeetingRoomWithExclusionModel()

	var buf bytes.Buffer
	err := goat.Debug(&buf, opts...)
	if err != nil {
		t.Fatalf("Debug failed: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(buf.Bytes(), &data); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	got, ok := data["summary"].(map[string]any)
	if !ok {
		t.Fatalf("Expected summary to be an object")
	}

	expectedSummary := map[string]any{
		"total_worlds": float64(10606),
		"invariant_violations": map[string]any{
			"found": false,
			"count": float64(0),
		},
	}

	ignoreOpts := cmpopts.IgnoreMapEntries(func(k string, _ any) bool {
		return k == "execution_time_ms"
	})

	if diff := cmp.Diff(expectedSummary, got, ignoreOpts); diff != "" {
		t.Errorf("Summary mismatch (-want +got):\n%s", diff)
	}

}
