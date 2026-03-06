package goat

import "reflect"

func deepCopyValue(v reflect.Value) reflect.Value {
	switch v.Kind() {
	case reflect.Map:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		newMap := reflect.MakeMapWithSize(v.Type(), v.Len())
		iter := v.MapRange()
		for iter.Next() {
			newMap.SetMapIndex(iter.Key(), deepCopyValue(iter.Value()))
		}
		return newMap
	case reflect.Slice:
		if v.IsNil() {
			return reflect.Zero(v.Type())
		}
		newSlice := reflect.MakeSlice(v.Type(), v.Len(), v.Len())
		for i := 0; i < v.Len(); i++ {
			newSlice.Index(i).Set(deepCopyValue(v.Index(i)))
		}
		return newSlice
	default:
		return v
	}
}

func deepCopyStructFields(v reflect.Value) {
	if v.Kind() != reflect.Struct {
		return
	}
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		if !field.CanSet() {
			continue
		}
		switch field.Kind() {
		case reflect.Map:
			field.Set(deepCopyValue(field))
		case reflect.Slice:
			field.Set(deepCopyValue(field))
		case reflect.Struct:
			deepCopyStructFields(field)
		}
	}
}

func cloneState(state AbstractState) AbstractState {
	v := reflect.ValueOf(state)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	newState := reflect.New(v.Type()).Elem()
	newState.Set(v)
	deepCopyStructFields(newState)

	return newState.Addr().Interface().(AbstractState)
}

func cloneStateMachine(sm AbstractStateMachine) AbstractStateMachine {
	v := reflect.ValueOf(sm)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	smc := reflect.New(v.Type()).Elem()

	smc.Set(v)
	eventHandlersField := smc.FieldByName("EventHandlers")
	oldHandlers := eventHandlersField.Interface().(map[AbstractState][]handlerInfo)

	// EventHandlers are copied separately below.
	// HandlerBuilders are safe to nil: NewInstance rebuilds them from spec,
	// and environment.clone (model checking) does not reference them.
	// Set both to nil to avoid unnecessary deep copy in deepCopyStructFields.
	smc.FieldByName("EventHandlers").Set(reflect.Zero(smc.FieldByName("EventHandlers").Type()))
	smc.FieldByName("HandlerBuilders").Set(reflect.Zero(smc.FieldByName("HandlerBuilders").Type()))

	deepCopyStructFields(smc)

	currentStateField := smc.FieldByName("State")
	if currentStateField.IsValid() && !currentStateField.IsZero() {
		state := currentStateField.Interface().(AbstractState)
		currentStateField.Set(reflect.ValueOf(cloneState(state)))
	}

	newHandlers := make(map[AbstractState][]handlerInfo, len(oldHandlers))

	for state, handlers := range oldHandlers {
		newState := cloneState(state)
		// NITS: this is a shallow copy,
		// but it's fine since we don't expect the handlers to be mutated
		newHandlers[newState] = append([]handlerInfo{}, handlers...)
	}

	eventHandlersField.Set(reflect.ValueOf(newHandlers))

	return smc.Addr().Interface().(AbstractStateMachine)
}

// WARNING: cloneEvent deep copies map and slice fields, but nested pointers are still shared.
func cloneEvent(event AbstractEvent) AbstractEvent {
	v := reflect.ValueOf(event)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	newEvent := reflect.New(v.Type()).Elem()
	newEvent.Set(v)
	deepCopyStructFields(newEvent)

	return newEvent.Addr().Interface().(AbstractEvent)
}
