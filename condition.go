package goat

import "strings"

// ConditionName represents the identifier for a condition.
type ConditionName string

func (c ConditionName) String() string {
	return strings.TrimSpace(string(c))
}

// Condition represents a named predicate evaluated against a world.
// Implementations must return true when the condition holds for the
// provided world, and false otherwise.
type Condition interface {
	Name() ConditionName
	Evaluate(w world) bool
}

type conditionFunc struct {
	name ConditionName
	fn   func(w world) bool
}

func (f conditionFunc) Name() ConditionName   { return f.name }
func (f conditionFunc) Evaluate(w world) bool { return f.fn(w) }

// BoolCondition creates a condition from a constant boolean value.
// This is useful for creating conditions that always pass (true) or always
// fail (false), typically used for testing or as placeholder conditions.
//
// Parameters:
//   - name: The name of this condition
//   - b: The boolean value that this condition will always return
//
// Returns a Condition that can be used with Test() (for example via WithRules(Always(...))).
//
// Example:
//
//	alwaysPass := goat.BoolCondition("pass", true)
//	alwaysFail := goat.BoolCondition("fail", false)
func BoolCondition(name string, b bool) Condition {
	return conditionFunc{name: ConditionName(name), fn: func(w world) bool { return b }}
}

// NewCondition creates a condition for a specific state machine instance.
// It allows checking properties of that particular state machine during
// model exploration and testing.
//
// Parameters:
//   - name: The condition name
//   - sm: The state machine instance to create a condition for
//   - check: A predicate function that returns true if the condition holds
//
// Returns a Condition that can be used with Test() (for example via WithRules(Always(...))).
//
// Example:
//
//	serverCond := goat.NewCondition("conn-limit", serverSM, func(sm *ServerStateMachine) bool {
//	    return sm.ConnectionCount <= sm.MaxConnections
//	})
func NewCondition[T AbstractStateMachine](name string, sm T, check func(T) bool) Condition {
	id := sm.id()
	return conditionFunc{name: ConditionName(name), fn: func(w world) bool {
		machine, exists := w.env.machines[id]
		if !exists {
			return false
		}
		typedMachine, ok := machine.(T)
		if !ok {
			return false
		}
		return check(typedMachine)
	}}
}

// Machines provides type-safe access to state machines during condition evaluation.
// It is used inside check functions to reference multiple state machines.
//
// Implementations return false when the requested state machine does not exist
// in the current world.
type Machines interface {
	Get(sm AbstractStateMachine) (AbstractStateMachine, bool)
}

type machinesImpl struct {
	world world
}

func (m *machinesImpl) Get(sm AbstractStateMachine) (AbstractStateMachine, bool) {
	id := sm.id()
	machine, exists := m.world.env.machines[id]
	if !exists {
		return nil, false
	}
	return machine, true
}

// GetMachine provides type-safe access to a state machine from Machines.
//
// Parameters:
//   - m: Machines accessor provided to the check function
//   - sm: A sample instance used to identify the target machine by ID
//
// Returns the typed state machine and true on success. Returns the zero value
// and false when the machine does not exist or the type does not match.
//
// Example:
//
//	machine, ok := goat.GetMachine(machines, client)
//	if !ok { return false }
//	state := machine.currentState().(*ClientState)
func GetMachine[T AbstractStateMachine](m Machines, sm T) (T, bool) {
	am, ok := m.Get(sm)
	if !ok {
		var zero T
		return zero, false
	}
	typed, ok := am.(T)
	if !ok {
		var zero T
		return zero, false
	}
	return typed, true
}

// NewMultiCondition creates a condition that can reference multiple state machines.
// The provided check function receives a Machines accessor.
//
// Parameters:
//   - name: The condition name
//   - checkFunc: Predicate that inspects one or more state machines
//   - sms: State machines referenced by the condition
//
// Returns a Condition that can be used with Test() (for example via WithRules(Always(...))).
//
// Example:
//
//	func NewConditionClientServer(client *Client, server *Server, check func(*Client, *Server) bool) goat.Condition {
//	    return goat.NewMultiCondition("client-server", func(machines goat.Machines) bool {
//	        c, ok1 := goat.GetMachine(machines, client)
//	        if !ok1 { return false }
//	        s, ok2 := goat.GetMachine(machines, server)
//	        if !ok2 { return false }
//
//	        return check(c, s)
//	    }, client, server)
//	}
//
//	// Usage
//	cond := NewConditionClientServer(client, server, func(c *Client, s *Server) bool {
//	    // business logic referencing both machines
//	    return c.Server != nil && s != nil
//	})
func NewMultiCondition(name string, checkFunc func(Machines) bool, sms ...AbstractStateMachine) Condition {
	return conditionFunc{name: ConditionName(name), fn: func(w world) bool {
		m := &machinesImpl{world: w}
		for _, sm := range sms {
			if _, ok := m.Get(sm); !ok {
				return false
			}
		}
		return checkFunc(m)
	}}
}

// NewCondition2 creates a condition that references two state machines.
//
// Parameters:
//   - name: The condition name
//   - sm1, sm2: The state machines to reference
//   - check: A predicate function that returns true if the condition holds
//
// Returns a Condition that can be used with Test() (for example via WithRules(Always(...))).
//
// Example:
//
//	cond := goat.NewCondition2("pair", client, server, func(c *Client, s *Server) bool {
//	    return true
//	})
func NewCondition2[T1, T2 AbstractStateMachine](name string, sm1 T1, sm2 T2, check func(T1, T2) bool) Condition {
	return NewMultiCondition(name, func(ms Machines) bool {
		m1, ok := GetMachine(ms, sm1)
		if !ok {
			return false
		}
		m2, ok := GetMachine(ms, sm2)
		if !ok {
			return false
		}
		return check(m1, m2)
	}, sm1, sm2)
}

// NewCondition3 creates a condition that references three state machines.
//
// Parameters:
//   - name: The condition name
//   - sm1, sm2, sm3: The state machines to reference
//   - check: A predicate function that returns true if the condition holds
//
// Returns a Condition that can be used with Test() (for example via WithRules(Always(...))).
//
// Example:
//
//	cond := goat.NewCondition3("triple", client, server, db, func(c *Client, s *Server, d *Database) bool {
//	    return true
//	})
func NewCondition3[T1, T2, T3 AbstractStateMachine](name string, sm1 T1, sm2 T2, sm3 T3, check func(T1, T2, T3) bool) Condition {
	return NewMultiCondition(name, func(ms Machines) bool {
		m1, ok := GetMachine(ms, sm1)
		if !ok {
			return false
		}
		m2, ok := GetMachine(ms, sm2)
		if !ok {
			return false
		}
		m3, ok := GetMachine(ms, sm3)
		if !ok {
			return false
		}
		return check(m1, m2, m3)
	}, sm1, sm2, sm3)
}
