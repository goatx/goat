package goat

// Rule represents a property that can be enforced during model checking.
// Rules register additional checks that run while exploring the state space.
type Rule interface {
	apply(*options)
}

type ruleFunc func(*options)

func (f ruleFunc) apply(o *options) {
	f(o)
}

// WithRules returns an Option that registers the provided rules during model checking.
//
// Parameters:
//   - rs: Rules created with helpers such as Always or WheneverPEventuallyQ
//
// Returns an Option that can be supplied to Test.
//
// Example:
//
//	result, err := goat.Test(
//		goat.WithStateMachines(node),
//		goat.WithRules(
//			goat.Always(consistency),
//			goat.WheneverPEventuallyQ(requested, responded),
//		),
//	)
func WithRules(rs ...Rule) Option {
	return optionFunc(func(o *options) {
		for _, r := range rs {
			if r == nil {
				continue
			}
			r.apply(o)
		}
	})
}

// Always returns a rule that ensures c holds in every explored world.
//
// Parameters:
//   - c: Condition that must remain true in all explored states
//
// Returns a Rule that can be supplied to WithRules.
//
// Example:
//
//	result, err := goat.Test(
//		goat.WithStateMachines(node),
//		goat.WithRules(
//			goat.Always(healthy),
//		),
//	)
func Always(c Condition) Rule {
	return ruleFunc(func(o *options) {
		registerCondition(o, c)
		o.invariants = append(o.invariants, c.Name())
	})
}

func registerCondition(o *options, c Condition) {
	if c == nil {
		return
	}
	if o.conds == nil {
		o.conds = make(map[ConditionName]Condition)
	}
	o.conds[c.Name()] = c
}

func registerTemporalRule(o *options, rule ltlRule) {
	o.ltlRules = append(o.ltlRules, rule)
}
