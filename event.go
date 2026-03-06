package goat

import (
	"fmt"
	"reflect"
	"strings"
)

type AbstractEvent interface {
	isEvent() bool
	setRoutingInfo(AbstractStateMachine, AbstractStateMachine)
}

// Event is the base struct that should be embedded in all event implementations.
// It provides the required methods to satisfy the AbstractEvent interface
// and ensures events are properly copyable for the state machine system.
//
// Example:
//
//	type MyCustomEvent struct {
//		goat.Event[*ProducerStateMachine, *ConsumerStateMachine]
//		Payload string
//	}
type Event[Sender AbstractStateMachine, Recipient AbstractStateMachine] struct {
	// this is needed to make Event copyable
	_ rune

	sender    Sender
	recipient Recipient
}

// UnTypedEvent is a convenience alias for an Event that does not specify
// concrete sender or recipient types.
type UnTypedEvent = Event[AbstractStateMachine, AbstractStateMachine]

func (*Event[Sender, Recipient]) isEvent() bool { return true }

// Sender returns the sender using the concrete type specified by the event's
// type parameters. If the actual sender does not match the requested type, the
// zero value for Sender is returned.
//
// Parameters:
//   - e: The event whose sender should be retrieved.
//
// Example:
//
//	var event goat.Event[*ProducerStateMachine, *ConsumerStateMachine]
//	typedSender := event.Sender()
//	if typedSender != nil {
//	    // Use the strongly typed sender
//	}
func (e *Event[Sender, Recipient]) Sender() Sender {
	if e == nil {
		var zero Sender
		return zero
	}
	return e.sender
}

// Recipient returns the recipient using the concrete type specified by the
// event's type parameters. If the actual recipient does not match the requested
// type, the zero value for Recipient is returned.
//
// Parameters:
//   - e: The event whose recipient should be retrieved.
//
// Example:
//
//	var event goat.Event[*ProducerStateMachine, *ConsumerStateMachine]
//	typedRecipient := event.Recipient()
//	if typedRecipient != nil {
//	    // Use the strongly typed recipient
//	}
func (e *Event[Sender, Recipient]) Recipient() Recipient {
	if e == nil {
		var zero Recipient
		return zero
	}
	return e.recipient
}

func (e *Event[Sender, Recipient]) setRoutingInfo(sender, recipient AbstractStateMachine) {
	if e == nil {
		return
	}

	if typed, ok := sender.(Sender); ok {
		e.sender = typed
	} else {
		var zero Sender
		e.sender = zero
	}

	if typed, ok := recipient.(Recipient); ok {
		e.recipient = typed
	} else {
		var zero Recipient
		e.recipient = zero
	}
}

type entryEvent struct {
	UnTypedEvent
}

type exitEvent struct {
	UnTypedEvent
}

type transitionEvent struct {
	UnTypedEvent
	To AbstractState
}

type haltEvent struct {
	UnTypedEvent
}

func newEventPrototype[T AbstractEvent]() AbstractEvent {
	var zero T
	eventType := reflect.TypeOf(zero)
	if eventType == nil {
		eventType = reflect.TypeFor[T]()
	}

	if eventType.Kind() == reflect.Interface {
		panic(fmt.Sprintf("cannot use interface type %s as event type parameter; use a concrete event type instead", eventType))
	}

	if eventType.Kind() == reflect.Pointer {
		elem := eventType.Elem()
		if elem.Kind() == reflect.Interface {
			panic(fmt.Sprintf("cannot use interface type %s as event type parameter; use a concrete event type instead", elem))
		}

		prototype := reflect.New(elem).Interface()
		evt, ok := prototype.(AbstractEvent)
		if !ok {
			panic(fmt.Sprintf("type %s does not implement AbstractEvent", eventType))
		}
		return evt
	}

	value := reflect.New(eventType).Elem().Interface()
	evt, ok := value.(AbstractEvent)
	if !ok {
		panic(fmt.Sprintf("type %s does not implement AbstractEvent", eventType))
	}
	return evt
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

		if fieldName != "Event" && fieldName != "UnTypedEvent" {
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
