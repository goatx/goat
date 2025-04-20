package goat

// Invariant represents a condition that must hold true throughout
// the state machine execution. It is used to verify system properties
// during architecture testing.
//
// Implementations must ensure that Evaluate returns true when the
// invariant holds for the given world state, and false otherwise.
type Invariant interface {
	Evaluate(w world) bool
}

type invariantFunc func(w world) bool

func (f invariantFunc) Evaluate(w world) bool {
	return f(w)
}

// BoolInvariant creates a simple invariant from a constant boolean value.
// This is useful for creating invariants that always pass (true) or always
// fail (false), typically used for testing or as placeholder invariants.
//
// Parameters:
//   - b: The boolean value that this invariant will always return
//
// Returns an Invariant that can be used with Test() or WithInvariants().
//
// Example:
//
//	alwaysPass := goat.BoolInvariant(true)
//	alwaysFail := goat.BoolInvariant(false)
func BoolInvariant(b bool) Invariant {
	return invariantFunc(func(w world) bool {
		return b
	})
}

// NewInvariant creates an invariant for a specific state machine instance.
// It allows checking properties of that particular state machine during
// model exploration and testing.
//
// Parameters:
//   - sm: The state machine instance to create an invariant for
//   - check: A predicate function that returns true if the invariant holds
//
// Returns an Invariant that can be used with Test() or WithInvariants().
//
// Example:
//
//	serverInv := goat.NewInvariant(serverSM, func(sm *ServerStateMachine) bool {
//	    return sm.ConnectionCount <= sm.MaxConnections
//	})
func NewInvariant[T AbstractStateMachine](sm T, check func(T) bool) Invariant {
	id := sm.id()
	return invariantFunc(func(w world) bool {
		machine, exists := w.env.machines[id]
		if !exists {
			return false
		}
		typedMachine, ok := machine.(T)
		if !ok {
			return false
		}
		return check(typedMachine)
	})
}
