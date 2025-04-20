package goat

type testStateMachine struct {
	StateMachine
}

type testState struct {
	State
	Name string
}

type testEvent struct {
	Event
	Value int
}

const testStateMachineID = "testStateMachine"

func newTestState(name string) *testState {
	return &testState{Name: name}
}

func newTestStateMachine(initialState AbstractState, states ...AbstractState) *testStateMachine {
	spec := NewStateMachineSpec(&testStateMachine{})
	allStates := append([]AbstractState{initialState}, states...)
	spec.DefineStates(allStates...)
	spec.SetInitialState(initialState)
	sm, err := spec.NewInstance()
	if err != nil {
		panic(err.Error()) // Test helper can panic for simplicity
	}
	return sm
}

func newTestEnvironment(machines ...*testStateMachine) environment {
	env := environment{
		machines: make(map[string]AbstractStateMachine),
		queue:    make(map[string][]AbstractEvent),
	}
	for _, sm := range machines {
		env.machines[sm.id()] = sm
	}
	return env
}

func newTestWorld(env environment) world {
	return newWorld(env)
}
