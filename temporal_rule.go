package goat

import "fmt"

type ltlRule struct {
	n string
	b *ba
}

func (r ltlRule) name() string { return r.n }
func (r ltlRule) ba() *ba      { return r.b }

type temporalEvidence interface {
	temporalEvidence()
}

type temporalRuleResult struct {
	Rule      string           `json:"rule"`
	Satisfied bool             `json:"satisfied"`
	Evidence  temporalEvidence `json:"evidence,omitempty"`
}

// WheneverPEventuallyQ returns a rule enforcing that whenever p holds, q eventually holds.
//
// Parameters:
//   - p: Condition that triggers the obligation
//   - q: Condition that must eventually become true
//
// Returns a Rule that can be registered with WithRules.
//
// Example:
//
//	result, err := goat.Test(
//		goat.WithStateMachines(primary, replica),
//		goat.WithRules(
//			goat.WheneverPEventuallyQ(write, replicated),
//		),
//	)
func WheneverPEventuallyQ(p, q Condition) Rule {
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

	return ruleFunc(func(o *options) {
		registerCondition(o, p)
		registerCondition(o, q)
		registerTemporalRule(o, ltlRule{n: name, b: b})
	})
}

// EventuallyAlways returns a rule enforcing that c eventually holds forever.
//
// Parameters:
//   - c: Condition that must eventually remain true
//
// Returns a Rule that can be registered with WithRules.
//
// Example:
//
//	result, err := goat.Test(
//		goat.WithStateMachines(nodes...),
//		goat.WithRules(
//			goat.EventuallyAlways(stable),
//		),
//	)
func EventuallyAlways(c Condition) Rule {
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

	return ruleFunc(func(o *options) {
		registerCondition(o, c)
		registerTemporalRule(o, ltlRule{n: name, b: b})
	})
}

// AlwaysEventually returns a rule enforcing that c holds infinitely often.
//
// Parameters:
//   - c: Condition that must recur indefinitely
//
// Returns a Rule that can be registered with WithRules.
//
// Example:
//
//	result, err := goat.Test(
//		goat.WithStateMachines(node),
//		goat.WithRules(
//			goat.AlwaysEventually(heartbeat),
//		),
//	)
func AlwaysEventually(c Condition) Rule {
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

	return ruleFunc(func(o *options) {
		registerCondition(o, c)
		registerTemporalRule(o, ltlRule{n: name, b: b})
	})
}
