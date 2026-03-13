package goat

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Test performs model checking on state machines with the provided options.
// It creates a Kripke model, explores all reachable states, checks invariants
// and temporal properties, and returns the results.
//
// Parameters:
//   - opts: Configuration options including state machines and rules
//
// Returns:
//   - *Result: verification results (violations, path information, summary)
//   - error: if model creation or solving fails
//
// Example:
//
//	result, err := goat.Test(
//	    goat.WithStateMachines(serverSM, clientSM),
//	    goat.WithRules(goat.Always(cond)),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// result is automatically printed to stdout
func Test(opts ...Option) (*Result, error) {
	model, err := newModel(opts...)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	if err := model.Solve(); err != nil {
		return nil, err
	}
	trResults := model.checkLTL()
	executionTime := time.Since(start).Milliseconds()

	result := model.buildResult(trResults, executionTime)
	_, _ = fmt.Fprint(os.Stdout, result)

	return result, nil
}

// WithStateMachines configures the test with the specified state machines.
// These state machines will be included in the model checking process.
//
// Parameters:
//   - sms: The state machines to include in the test
//
// Returns an Option that can be passed to Test() or Debug().
//
// Example:
//
//	goat.WithStateMachines(serverSM, clientSM, proxysSM)
func WithStateMachines(sms ...AbstractStateMachine) Option {
	return optionFunc(func(o *options) {
		o.sms = sms
	})
}

// Debug performs model checking and outputs detailed JSON results.
// Unlike Test(), this function provides comprehensive debugging information
// including all explored worlds and their states in JSON format.
//
// Parameters:
//   - w: Writer to output the JSON results to
//   - opts: Configuration options including state machines and invariants
//
// Returns an error if model creation, solving, or JSON encoding fails.
//
// Example:
//
//	var buf bytes.Buffer
//	err := goat.Debug(&buf, goat.WithStateMachines(sm), goat.WithRules(goat.Always(cond)))
//	fmt.Println(buf.String()) // JSON output
func Debug(w io.Writer, opts ...Option) error {
	model, err := newModel(opts...)
	if err != nil {
		return err
	}

	start := time.Now()
	if err := model.Solve(); err != nil {
		return err
	}
	executionTime := time.Since(start).Milliseconds()

	worlds := model.worldsToJSON()
	summary := model.summarize(executionTime)
	temporal := model.checkLTL()

	result := map[string]any{
		"worlds":  worlds,
		"summary": summary,
	}
	if len(temporal) > 0 {
		result["temporal_rules"] = temporal
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// WriteDot performs model checking and outputs the state graph in DOT format.
// The output can be used with Graphviz to visualize the state space and
// identify paths to invariant violations.
//
// Parameters:
//   - w: Writer to output the DOT graph to
//   - opts: Configuration options including state machines and invariants
//
// Returns an error if model creation or solving fails.
//
// Example:
//
//		var file *os.File
//		file, err := os.Create("model.dot")
//		if err != nil {
//		    return err
//		}
//		defer file.Close()
//	     err = goat.WriteDot(file, goat.WithStateMachines(sm), goat.WithRules(goat.Always(cond)))
func WriteDot(w io.Writer, opts ...Option) error {
	model, err := newModel(opts...)
	if err != nil {
		return err
	}

	if err := model.Solve(); err != nil {
		return err
	}

	model.writeDot(w)
	return nil
}
