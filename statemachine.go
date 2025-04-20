package goat

import (
	"fmt"
	"reflect"
	"strings"
)

const noFieldsMessage = "no fields"

type handlerBuilder func(smID string) handler

type handlerBuilderInfo struct {
	event   AbstractEvent
	builder handlerBuilder
}

// StateMachineSpec defines the specification for a state machine type.
// It serves as a template for creating multiple instances of the same
// state machine with consistent behavior and state definitions.
//
// Use NewStateMachineSpec to create a specification, then configure it
// with DefineStates and SetInitialState before creating instances.
type StateMachineSpec[T AbstractStateMachine] struct {
	prototype       T
	states          []AbstractState
	initialState    AbstractState
	handlerBuilders map[AbstractState][]handlerBuilderInfo
}

// NewStateMachineSpec creates a new state machine specification with
// the given prototype. The prototype defines the structure and behavior
// that all instances created from this spec will share.
//
// Parameters:
//   - prototype: An instance of the state machine type to use as template
//
// Returns a specification that can be configured with states and handlers.
//
// Example:
//
//	spec := goat.NewStateMachineSpec(&MyStateMachine{})
//	spec.DefineStates(StateA{}, StateB{}).
//	     SetInitialState(StateA{})
func NewStateMachineSpec[T AbstractStateMachine](prototype T) *StateMachineSpec[T] {
	return &StateMachineSpec[T]{
		prototype:       prototype,
		handlerBuilders: make(map[AbstractState][]handlerBuilderInfo),
	}
}

// DefineStates sets the valid states for this state machine specification.
// It configures all provided states and sets up default handlers for each.
//
// Parameters:
//   - states: All valid states that this state machine can be in
//
// Returns the spec for method chaining.
//
// Example:
//
//	spec.DefineStates(IdleState{}, ActiveState{}, ErrorState{})
func (spec *StateMachineSpec[T]) DefineStates(states ...AbstractState) *StateMachineSpec[T] {
	spec.states = states
	for _, state := range states {
		spec.setDefaultHandlerBuilders(state)
	}
	return spec
}

// SetInitialState defines which state new instances will start in.
// The provided state must be one of the states defined in DefineStates.
//
// Parameters:
//   - state: The state that new instances should start in
//
// Returns the spec for method chaining.
//
// Example:
//
//	spec.SetInitialState(IdleState{})
func (spec *StateMachineSpec[T]) SetInitialState(state AbstractState) *StateMachineSpec[T] {
	spec.initialState = state
	return spec
}

func (spec *StateMachineSpec[T]) setDefaultHandlerBuilders(state AbstractState) {
	transitionBuilder := func(smID string) handler {
		return &defaultOnTransitionHandler{}
	}
	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &transitionEvent{},
		builder: transitionBuilder,
	})

	haltBuilder := func(smID string) handler {
		return &defaultOnHaltHandler{}
	}
	spec.handlerBuilders[state] = append(spec.handlerBuilders[state], handlerBuilderInfo{
		event:   &haltEvent{},
		builder: haltBuilder,
	})
}

func (spec *StateMachineSpec[T]) validate() error {
	if spec.initialState == nil {
		return fmt.Errorf("state machine spec has no initial state")
	}

	for _, definedState := range spec.states {
		if sameState(definedState, spec.initialState) {
			return nil
		}
	}

	return fmt.Errorf("initial state is not in defined states")
}

// NewInstance creates a new state machine instance based on this specification.
// Each instance is independent and starts in the initial state defined by
// SetInitialState with all handlers configured.
//
// Returns a fully configured state machine instance and any validation error.
//
// Example:
//
//	instance1, err := spec.NewInstance()
//	if err != nil {
//		return err
//	}
//	instance2, err := spec.NewInstance() // Independent instance
func (spec *StateMachineSpec[T]) NewInstance() (T, error) {
	var zero T
	if err := spec.validate(); err != nil {
		return zero, err
	}

	instance := cloneStateMachine(spec.prototype).(T)
	innerSM := getInnerStateMachine(instance)

	innerSM.smID = getStateMachineName(instance)
	innerSM.EventHandlers = nil // Will be built later in initialWorld
	innerSM.HandlerBuilders = make(map[AbstractState][]handlerBuilderInfo)
	innerSM.State = spec.initialState
	innerSM.halted = false

	for state, builders := range spec.handlerBuilders {
		innerSM.HandlerBuilders[state] = append([]handlerBuilderInfo{}, builders...)
	}

	return instance, nil
}

// AbstractState is the base interface for all states in the state machine.
// States represent discrete conditions or modes that a state machine can be in.
//
// Implementations should embed the State struct to satisfy this interface.
//
// Example:
//
//	type IdleState struct {
//	    goat.State
//	    Timeout int
//	}
type AbstractState interface {
	isState() bool
}

// State is the base struct that should be embedded in all state implementations.
// It provides the required methods to satisfy the AbstractState interface
// and ensures states are properly copyable for the state machine system.
//
// Example:
//
//	type MyState struct {
//	    goat.State
//	    CustomField int
//	}
type State struct {
	// this is needed to make State copyable
	_ rune
}

func (*State) isState() bool {
	return true
}

func cloneState(state AbstractState) AbstractState {
	v := reflect.ValueOf(state)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	newState := reflect.New(v.Type()).Elem()
	newState.Set(v)

	return newState.Addr().Interface().(AbstractState)
}

func sameState(s1, s2 AbstractState) bool {
	return getStateDetails(s1) == getStateDetails(s2)
}

// AbstractStateMachine is the base interface for all state machines.
// State machines encapsulate behavior and state transitions in response to events.
//
// Implementations must embed the StateMachine struct. Instances are typically
// created through NewStateMachineSpec and its methods for proper configuration.
//
// Example:
//
//	type MyStateMachine struct {
//	    goat.StateMachine
//	    CustomData string
//	}
//
//	// Create via specification
//	spec := goat.NewStateMachineSpec(&MyStateMachine{})
//	spec.DefineStates(IdleState{}, ActiveState{}).
//	     SetInitialState(IdleState{})
//	instance := spec.NewInstance()
type AbstractStateMachine interface {
	isStateMachine() bool
	currentState() AbstractState
	setCurrentState(state AbstractState)
	id() string
	SetInitialState(state AbstractState)
}

// StateMachine provides the core infrastructure for state machine behavior.
// It must be embedded in concrete state machine implementations to provide
// state management, event handling, and transition capabilities.
//
// Do not instantiate directly; instead embed in your state machine types
// and use NewStateMachineSpec to create properly configured instances.
//
// Example:
//
//	type MyStateMachine struct {
//	    goat.StateMachine
//	    Data string
//	}
type StateMachine struct {
	smID            string
	EventHandlers   map[AbstractState][]handlerInfo
	HandlerBuilders map[AbstractState][]handlerBuilderInfo
	halted          bool
	State           AbstractState
}

func cloneStateMachine(sm AbstractStateMachine) AbstractStateMachine {
	v := reflect.ValueOf(sm)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	smc := reflect.New(v.Type()).Elem()

	smc.Set(v)

	currentStateField := smc.FieldByName("State")
	if currentStateField.IsValid() && !currentStateField.IsZero() {
		state := currentStateField.Interface().(AbstractState)
		currentStateField.Set(reflect.ValueOf(cloneState(state)))
	}

	eventHandlersField := smc.FieldByName("EventHandlers")
	if eventHandlersField.IsValid() && !eventHandlersField.IsZero() {
		oldHandlers := eventHandlersField.Interface().(map[AbstractState][]handlerInfo)
		newHandlers := make(map[AbstractState][]handlerInfo, len(oldHandlers))

		for state, handlers := range oldHandlers {
			newState := cloneState(state)
			// NITS: this is a shallow copy,
			// but it's fine since we don't expect the handlers to be mutated
			newHandlers[newState] = append([]handlerInfo{}, handlers...)
		}

		eventHandlersField.Set(reflect.ValueOf(newHandlers))
	}

	return smc.Addr().Interface().(AbstractStateMachine)
}

func (*StateMachine) isStateMachine() bool {
	return true
}

func (sm *StateMachine) currentState() AbstractState {
	return sm.State
}

func (sm *StateMachine) setCurrentState(state AbstractState) {
	sm.State = state
}

func (sm *StateMachine) id() string {
	if sm.smID == "" {
		panic("please call New() before anything else")
	}
	return sm.smID
}

// SetInitialState sets the starting state for this state machine instance.
// This can only be called once per instance, typically during initialization.
//
// Parameters:
//   - state: The state this state machine should start in
//
// Panics if the initial state has already been set.
//
// Example:
//
//	sm.SetInitialState(&IdleState{})
func (sm *StateMachine) SetInitialState(state AbstractState) {
	if sm.State != nil {
		panic("initial state already set")
	}
	sm.State = state
}

func getInnerStateMachine(sm AbstractStateMachine) *StateMachine {
	v := reflect.ValueOf(sm)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID STATE MACHINE: %v", sm))
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	field := v.FieldByName("StateMachine")
	if field.IsValid() && field.CanAddr() {
		innerSm, ok := field.Addr().Interface().(*StateMachine)
		if ok {
			return innerSm
		}
	}
	panic("INVALID STATE MACHINE")
}

func getStateMachineName(sm AbstractStateMachine) string {
	v := reflect.ValueOf(sm)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID STATE MACHINE: %v", sm))
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	return v.Type().Name()
}

func getStateMachineDetails(sm AbstractStateMachine) string {
	v := reflect.ValueOf(sm)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID STATE MACHINE: %v", sm))
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()

	var fieldDetails []string
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name

		if field.Kind() == reflect.Ptr {
			continue
		}

		if fieldName != "StateMachine" {
			if field.CanInterface() {
				fieldType := field.Type().String()
				fieldValue := field.Interface()
				fieldDetails = append(fieldDetails, fmt.Sprintf("{Name:%s,Type:%s,Value:%v}", fieldName, fieldType, fieldValue))
			} else {
				fieldDetails = append(fieldDetails, fmt.Sprintf("{Name:%s,Type:%s,Value:[UNACCESSIBLE]}", fieldName, field.Type().String()))
			}
		}
	}

	if len(fieldDetails) == 0 {
		return noFieldsMessage
	}

	return strings.Join(fieldDetails, ",")
}

func getStateDetails(s AbstractState) string {
	v := reflect.ValueOf(s)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID STATE: %v", s))
	}

	t := v.Type()
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		t = t.Elem()
	}

	var fieldDetails []string
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name

		if fieldName != "State" {
			if field.CanInterface() {
				fieldType := field.Type().String()
				fieldValue := field.Interface()
				fieldDetails = append(fieldDetails, fmt.Sprintf("{Name:%s,Type:%s,Value:%v}", fieldName, fieldType, fieldValue))
			} else {
				fieldDetails = append(fieldDetails, fmt.Sprintf("{Name:%s,Type:%s,Value:[UNACCESSIBLE]}", fieldName, field.Type().String()))
			}
		}
	}

	if len(fieldDetails) == 0 {
		return noFieldsMessage
	}

	return strings.Join(fieldDetails, ",")
}
