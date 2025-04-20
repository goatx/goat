package goat

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// Test performs model checking on state machines with the provided options.
// It creates a Kripke model, explores all reachable states, checks invariants,
// and outputs results to stdout.
//
// Parameters:
//   - opts: Configuration options including state machines and invariants
//
// Returns an error if model creation, solving, or invariant checking fails.
//
// Example:
//
//	err := goat.Test(
//	    goat.WithStateMachines(serverSM, clientSM),
//	    goat.WithInvariants(safetyInvariant),
//	)
func Test(opts ...Option) error {
	model, err := newModel(opts...)
	if err != nil {
		return err
	}

	start := time.Now()
	if err := model.Solve(); err != nil {
		return err
	}
	executionTime := time.Since(start).Milliseconds()

	model.writeLog(os.Stdout, "invariant violation")

	summary := model.summarize(executionTime)
	_, _ = fmt.Fprintln(os.Stdout, "\nModel Checking Summary:")
	_, _ = fmt.Fprintf(os.Stdout, "Total Worlds: %d\n", summary.TotalWorlds)
	if summary.InvariantViolations.Found {
		_, _ = fmt.Fprintf(os.Stdout, "Invariant Violations: %d found\n", summary.InvariantViolations.Count)
	} else {
		_, _ = fmt.Fprintln(os.Stdout, "Invariant Violations: None")
	}
	_, _ = fmt.Fprintf(os.Stdout, "Execution Time: %dms\n", summary.ExecutionTimeMs)

	return nil
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

// WithInvariants configures the test with the specified invariants.
// These invariants will be checked during model exploration to detect
// violations of system properties.
//
// Parameters:
//   - is: The invariants to check during testing
//
// Returns an Option that can be passed to Test() or Debug().
//
// Example:
//
//	goat.WithInvariants(exclusionInvariant, livenessInvariant)
func WithInvariants(is ...Invariant) Option {
	return optionFunc(func(o *options) {
		o.invariants = is
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
//	err := goat.Debug(&buf, goat.WithStateMachines(sm), goat.WithInvariants(inv))
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

	result := map[string]any{
		"worlds":  worlds,
		"summary": summary,
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
//	var file *os.File
//	file, err := os.Create("model.dot")
//	if err != nil {
//	    return err
//	}
//	defer file.Close()
//	err = goat.WriteDot(file, goat.WithStateMachines(sm), goat.WithInvariants(inv))
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
