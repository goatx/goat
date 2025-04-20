package goat

import (
	"fmt"
	"reflect"
	"strings"
)

// AbstractEvent is the base interface for all events in the state machine.
// Events trigger state transitions and handler executions, representing
// things that happen to or within the state machine.
//
// Most users will work with the built-in event types (entryEvent, exitEvent, etc.)
// or create simple custom events for specific scenarios.
type AbstractEvent interface {
	isEvent() bool
}

// Event is the base struct that should be embedded in all event implementations.
// It provides the required methods to satisfy the AbstractEvent interface
// and ensures events are properly copyable for the state machine system.
//
// Example:
//
//	type MyCustomEvent struct {
//	    Event
//	    Payload string
//	}
type Event struct {
	// this is needed to make Event copyable
	_ rune
}

func (*Event) isEvent() bool {
	return true
}

type entryEvent struct {
	Event
}

type exitEvent struct {
	Event
}

type transitionEvent struct {
	Event
	To AbstractState
}

type haltEvent struct {
	Event
}

// WARNING: cloneEvent performs shallow copy, so nested pointers are shared
// This is a potential bug - modifications to nested structs will affect both instances
func cloneEvent(event AbstractEvent) AbstractEvent {
	v := reflect.ValueOf(event)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	newEvent := reflect.New(v.Type()).Elem()
	newEvent.Set(v)

	return newEvent.Addr().Interface().(AbstractEvent)
}

func sameEvent(e1, e2 AbstractEvent) bool {
	return getEventName(e1) == getEventName(e2)
}

func getEventName(e AbstractEvent) string {
	v := reflect.ValueOf(e)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID EVENT: %v", e))
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	return v.Type().Name()
}

func getEventDetails(e AbstractEvent) string {
	v := reflect.ValueOf(e)
	if !v.IsValid() {
		panic(fmt.Sprintf("INVALID EVENT: %v", e))
	}
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	t := v.Type()

	var fieldDetails []string
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := t.Field(i).Name

		// WARNING: Pointer fields are intentionally skipped
		// This prevents potential circular references and nil pointer dereferences,
		// but means that important data in pointer fields will not be displayed
		if field.Kind() == reflect.Ptr {
			continue
		}

		if fieldName != "Event" {
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
