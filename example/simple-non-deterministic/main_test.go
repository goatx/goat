package main

import (
	"testing"

	"github.com/goatx/goat"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSimpleNonDeterministic(t *testing.T) {
	opts := createSimpleNonDeterministicModel()

	result, err := goat.Test(opts...)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	expected := &goat.Result{
		Summary: goat.Summary{TotalWorlds: 12},
	}

	cmpOpts := cmp.Options{
		cmpopts.IgnoreFields(goat.Summary{}, "ExecutionTimeMs"),
	}
	if diff := cmp.Diff(expected, result, cmpOpts...); diff != "" {
		t.Errorf("result mismatch (-want +got):\n%s", diff)
	}
}
