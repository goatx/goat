package goat

import (
	"fmt"
	"sort"
	"strings"
)

// Result holds the outcome of a model checking run.
type Result struct {
	Violations []Violation
	Summary    Summary
}

// HasViolation reports whether any violations were found.
func (r *Result) HasViolation() bool {
	return len(r.Violations) > 0
}

// String returns a human-readable report of the model checking results.
func (r *Result) String() string {
	var sb strings.Builder

	var invariants, temporals []Violation
	for _, v := range r.Violations {
		if v.Loop == nil {
			invariants = append(invariants, v)
		} else {
			temporals = append(temporals, v)
		}
	}

	if len(invariants) > 0 {
		writeInvariantViolations(&sb, invariants)
	}
	if len(temporals) > 0 {
		writeTemporalViolations(&sb, temporals)
	}
	if len(invariants) == 0 && len(temporals) == 0 {
		sb.WriteString("No violations found.\n")
	}

	fmt.Fprintln(&sb, "\nModel Checking Summary:")
	fmt.Fprintf(&sb, "Total Worlds: %d\n", r.Summary.TotalWorlds)
	fmt.Fprintf(&sb, "Execution Time: %dms\n", r.Summary.ExecutionTimeMs)

	return sb.String()
}

// Summary contains statistics about the model checking run.
type Summary struct {
	// TotalWorlds is the number of distinct worlds (unique combinations of
	// state machine states and queued events) explored during model checking.
	TotalWorlds     int
	ExecutionTimeMs int64
}

// Violation represents a single property violation found during model checking.
type Violation struct {
	Rule string
	Path []WorldSnapshot
	Loop []WorldSnapshot
}

// WorldSnapshot represents a world — the combination of every state machine's
// current state and all queued events at a single point in time.
// A violation path is a sequence of worlds that leads to the violation.
type WorldSnapshot struct {
	StateMachines []StateMachineSnapshot
	QueuedEvents  []EventSnapshot
}

// StateMachineSnapshot is a snapshot of a single state machine.
type StateMachineSnapshot struct {
	Name    string
	State   string
	Details string
}

// EventSnapshot is a snapshot of a queued event.
type EventSnapshot struct {
	TargetMachine string
	EventName     string
	Details       string
}

func (m *model) buildResult(trResults []temporalRuleResult, executionTimeMs int64) *Result {
	result := &Result{
		Summary: Summary{
			TotalWorlds:     len(m.worlds),
			ExecutionTimeMs: executionTimeMs,
		},
	}

	if m.hasInvariantViolation {
		for _, w := range m.collectInvariantViolations() {
			name := w.condition.String()
			rule := "Always " + name
			if name == "" {
				rule = ""
			}
			result.Violations = append(result.Violations, Violation{
				Rule: rule,
				Path: m.buildWorldSnapshots(w.path),
			})
		}
	}

	for _, tr := range trResults {
		if tr.Satisfied {
			continue
		}
		l, ok := tr.Evidence.(*lasso)
		if !ok || l == nil {
			continue
		}
		result.Violations = append(result.Violations, Violation{
			Rule: tr.Rule,
			Path: m.buildWorldSnapshots(l.Prefix),
			Loop: m.buildWorldSnapshots(l.Loop),
		})
	}

	return result
}

func (m *model) buildWorldSnapshots(ids []worldID) []WorldSnapshot {
	snapshots := make([]WorldSnapshot, len(ids))
	for i, wid := range ids {
		w := m.worlds[wid]
		snapshots[i] = m.buildWorldSnapshot(w)
	}
	return snapshots
}

func (*model) buildWorldSnapshot(w world) WorldSnapshot {
	smIDs := make([]string, 0, len(w.env.machines))
	for smID := range w.env.machines {
		smIDs = append(smIDs, smID)
	}
	sort.Strings(smIDs)

	sms := make([]StateMachineSnapshot, 0, len(smIDs))
	for _, smID := range smIDs {
		sm := w.env.machines[smID]
		sms = append(sms, StateMachineSnapshot{
			Name:    getStateMachineName(sm),
			State:   getStateDetails(sm.currentState()),
			Details: getStateMachineDetails(sm),
		})
	}

	events := make([]EventSnapshot, 0)
	for _, smID := range smIDs {
		if evts, ok := w.env.queue[smID]; ok {
			for _, evt := range evts {
				events = append(events, EventSnapshot{
					TargetMachine: getStateMachineName(w.env.machines[smID]),
					EventName:     getEventName(evt),
					Details:       getEventDetails(evt),
				})
			}
		}
	}

	return WorldSnapshot{
		StateMachines: sms,
		QueuedEvents:  events,
	}
}
