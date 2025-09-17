package goat

import "fmt"

// TemporalRule represents a temporal property specified by a Büchi automaton.
type TemporalRule interface {
	name() string
	ba() *ba
}

type temporalRule struct {
	n string
	b *ba
}

func (r temporalRule) name() string { return r.n }
func (r temporalRule) ba() *ba      { return r.b }

type temporalRuleResult struct {
	Rule  string `json:"rule"`
	Holds bool   `json:"holds"`
	Lasso *lasso `json:"lasso,omitempty"`
}

// WithTemporalRules registers temporal rules for model checking.
//
// Parameters:
//   - rs: The temporal rules to enforce during verification
//
// Returns an Option that can be passed to Test or Debug.
//
// Example:
//
//	err := goat.Test(
//	    goat.WithStateMachines(sm),
//	    goat.WithTemporalRules(goat.EventuallyAlways(cond)),
//	)
func WithTemporalRules(rs ...TemporalRule) Option {
	return optionFunc(func(o *options) {
		o.ltlRules = append(o.ltlRules, rs...)
	})
}

// ---------- Temporal rule constructors ----------

// WheneverPEventuallyQ returns a rule enforcing that whenever p holds, q eventually holds.
//
// Parameters:
//   - p: Condition that triggers the obligation
//   - q: Condition that must eventually become true
//
// Returns a TemporalRule that can be registered with WithTemporalRules.
//
// Example:
//
//	rule := goat.WheneverPEventuallyQ(write, replicated)
//	err := goat.Test(
//	    goat.WithStateMachines(primary, replica),
//	    goat.WithConditions(write, replicated),
//	    goat.WithTemporalRules(rule),
//	)
func WheneverPEventuallyQ(p, q Condition) TemporalRule {
	name := fmt.Sprintf("whenever %s eventually %s", p.Name(), q.Name())
	b := &ba{
		initial:   0,
		accepting: map[baState]bool{1: true},
		trans: map[baState][]baTransition{
			0: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return l[p.Name()] && !l[q.Name()] }},
				{to: 0, cond: func(l map[ConditionName]bool) bool { return !l[p.Name()] || l[q.Name()] }},
			},
			1: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return !l[q.Name()] }},
				{to: 2, cond: func(l map[ConditionName]bool) bool { return l[q.Name()] }},
			},
			2: {
				{to: 2, cond: func(map[ConditionName]bool) bool { return true }},
			},
		},
	}
	return temporalRule{n: name, b: b}
}

// EventuallyAlways returns a rule enforcing that c eventually holds forever.
//
// Parameters:
//   - c: Condition that must eventually remain true
//
// Returns a TemporalRule that can be registered with WithTemporalRules.
//
// Example:
//
//	rule := goat.EventuallyAlways(stable)
//	err := goat.Test(
//	    goat.WithStateMachines(nodes...),
//	    goat.WithConditions(stable),
//	    goat.WithTemporalRules(rule),
//	)
func EventuallyAlways(c Condition) TemporalRule {
	name := fmt.Sprintf("eventually always %s", c.Name())
	b := &ba{
		initial:   0,
		accepting: map[baState]bool{1: true},
		trans: map[baState][]baTransition{
			0: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return !l[c.Name()] }},
				{to: 0, cond: func(l map[ConditionName]bool) bool { return l[c.Name()] }},
			},
			1: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return !l[c.Name()] }},
				{to: 0, cond: func(l map[ConditionName]bool) bool { return l[c.Name()] }},
			},
		},
	}
	return temporalRule{n: name, b: b}
}

// AlwaysEventually returns a rule enforcing that c holds infinitely often.
//
// Parameters:
//   - c: Condition that must recur indefinitely
//
// Returns a TemporalRule that can be registered with WithTemporalRules.
//
// Example:
//
//	rule := goat.AlwaysEventually(heartbeat)
//	err := goat.Test(
//	    goat.WithStateMachines(node),
//	    goat.WithConditions(heartbeat),
//	    goat.WithTemporalRules(rule),
//	)
func AlwaysEventually(c Condition) TemporalRule {
	name := fmt.Sprintf("always eventually %s", c.Name())
	b := &ba{
		initial:   0,
		accepting: map[baState]bool{1: true},
		trans: map[baState][]baTransition{
			0: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return !l[c.Name()] }},
				{to: 0, cond: func(map[ConditionName]bool) bool { return true }},
			},
			1: {
				{to: 1, cond: func(l map[ConditionName]bool) bool { return !l[c.Name()] }},
				{to: 2, cond: func(l map[ConditionName]bool) bool { return l[c.Name()] }},
			},
			2: {
				{to: 2, cond: func(map[ConditionName]bool) bool { return true }},
			},
		},
	}
	return temporalRule{n: name, b: b}
}
