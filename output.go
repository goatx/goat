package goat

import (
	"fmt"
	"io"
	"reflect"
	"slices"
	"sort"
	"strings"
)

type modelSummary struct {
	TotalWorlds     int   `json:"total_worlds"`
	ExecutionTimeMs int64 `json:"execution_time_ms"`
}

func (m *model) writeDot(w io.Writer) {
	var sb strings.Builder

	sb.WriteString("digraph {\n")

	// ---------- Nodes ----------
	worldIDs := make([]worldID, 0, len(m.worlds))
	for id := range m.worlds {
		worldIDs = append(worldIDs, id)
	}
	sort.Slice(worldIDs, func(i, j int) bool { return worldIDs[i] < worldIDs[j] })

	for _, id := range worldIDs {
		wld := m.worlds[id]
		sb.WriteString("  ")
		fmt.Fprintf(&sb, "%d", id)
		sb.WriteString(` [ label="`)
		sb.WriteString(wld.label())
		sb.WriteString("\" ];\n")
		if id == m.initial.id {
			sb.WriteString("  ")
			fmt.Fprintf(&sb, "%d", id)
			sb.WriteString(" [ penwidth=5 ];\n")
		}
		if len(wld.failedInvariants) > 0 {
			sb.WriteString("  ")
			fmt.Fprintf(&sb, "%d", id)
			sb.WriteString(" [ color=red, penwidth=3 ];\n")
		}
	}

	// ---------- Edges ----------
	fromIDs := make([]worldID, 0, len(m.accessible))
	for from := range m.accessible {
		fromIDs = append(fromIDs, from)
	}
	sort.Slice(fromIDs, func(i, j int) bool { return fromIDs[i] < fromIDs[j] })

	for _, from := range fromIDs {
		tos := m.accessible[from]
		sort.Slice(tos, func(i, j int) bool { return tos[i] < tos[j] })
		fromStr := fmt.Sprintf("%d", from)
		for _, to := range tos {
			sb.WriteString("  ")
			sb.WriteString(fromStr)
			sb.WriteString(" -> ")
			fmt.Fprintf(&sb, "%d", to)
			sb.WriteString(";\n")
		}
	}
	sb.WriteString("}\n")

	// ------------ Output ------------
	_, _ = io.WriteString(w, sb.String())
}

func writeInvariantViolations(w io.Writer, violations []Violation) {
	var sb strings.Builder
	for i, v := range violations {
		if i > 0 {
			sb.WriteString("\n")
		}

		if v.Rule == "" {
			sb.WriteString("Condition failed.\n")
		} else {
			sb.WriteString("Condition failed. Not ")
			sb.WriteString(v.Rule)
			sb.WriteString(".\n")
		}

		sb.WriteString("Path (length = ")
		fmt.Fprintf(&sb, "%d", len(v.Path))
		sb.WriteString("):\n")

		pathLen := len(v.Path)
		writeWorldSequence(&sb, v.Path, func(idx int) string {
			if idx == pathLen-1 {
				return "<-- violation here"
			}
			return ""
		})
	}

	_, _ = io.WriteString(w, sb.String())
}

func writeTemporalViolations(w io.Writer, violations []Violation) {
	var sb strings.Builder
	block := 0

	for _, v := range violations {
		if block > 0 {
			sb.WriteString("\n")
		}
		block++

		rule := strings.TrimSpace(v.Rule)
		if rule == "" {
			sb.WriteString("Condition failed.\n")
		} else {
			sb.WriteString("Condition failed. Not ")
			sb.WriteString(rule)
			sb.WriteString(".\n")
		}

		prefixLen := len(v.Path)
		loopLen := len(v.Loop)

		sequence := make([]WorldSnapshot, 0, prefixLen+loopLen)
		sequence = append(sequence, v.Path...)
		if loopLen > 0 {
			if prefixLen == 0 || !reflect.DeepEqual(v.Path[prefixLen-1], v.Loop[0]) {
				sequence = append(sequence, v.Loop...)
			} else {
				sequence = append(sequence, v.Loop[1:]...)
			}
		}

		if len(sequence) == 0 {
			sb.WriteString("Violation path (length = 0):\n")
			sb.WriteString("  <empty witness>\n")
			continue
		}

		sb.WriteString("Violation path (length = ")
		fmt.Fprintf(&sb, "%d", len(sequence))
		sb.WriteString("):\n")

		writeWorldSequence(&sb, sequence, nil)
	}

	if block == 0 {
		return
	}

	_, _ = io.WriteString(w, sb.String())
}

func writeWorldSequence(sb *strings.Builder, snapshots []WorldSnapshot, annotate func(int) string) {
	for idx, snap := range snapshots {
		sb.WriteString("  [")
		fmt.Fprintf(sb, "%d", idx)
		sb.WriteString("]")
		if annotate != nil {
			if annotation := annotate(idx); annotation != "" {
				sb.WriteString(" ")
				sb.WriteString(annotation)
			}
		}
		sb.WriteString("\n")
		sb.WriteString("  StateMachines:\n")
		for _, sm := range snap.StateMachines {
			sb.WriteString("    Name: ")
			sb.WriteString(sm.Name)
			sb.WriteString(", Detail: ")
			sb.WriteString(sm.Details)
			sb.WriteString(", State: ")
			sb.WriteString(sm.State)
			sb.WriteString("\n")
		}
		sb.WriteString("  QueuedEvents:\n")
		for _, ev := range snap.QueuedEvents {
			sb.WriteString("    StateMachine: ")
			sb.WriteString(ev.TargetMachine)
			sb.WriteString(", Event: ")
			sb.WriteString(ev.EventName)
			sb.WriteString(", Detail: ")
			sb.WriteString(ev.Details)
			sb.WriteString("\n")
		}
	}
}

type invariantViolationWitness struct {
	path      []worldID
	condition ConditionName
}

func (m *model) collectInvariantViolations() []invariantViolationWitness {
	var violations []invariantViolationWitness

	targets := make(map[string]struct{})
	for _, world := range m.worlds {
		for _, name := range world.failedInvariants {
			targets[name.String()] = struct{}{}
		}
	}

	totalTargets := len(targets)
	if totalTargets == 0 {
		return nil
	}

	visited := make(map[worldID]bool)
	seen := make(map[string]bool)

	queue := [][]worldID{{m.initial.id}}

	for len(queue) > 0 {
		if len(seen) == totalTargets {
			break
		}

		path := queue[0]
		queue = queue[1:]

		currentID := path[len(path)-1]

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		world := m.worlds[currentID]

		if len(world.failedInvariants) > 0 {
			for _, name := range world.failedInvariants {
				if seen[name.String()] {
					continue
				}
				seen[name.String()] = true

				copyPath := slices.Clone(path)
				violations = append(violations, invariantViolationWitness{
					path:      copyPath,
					condition: name,
				})
			}
		}

		for _, nextID := range m.accessible[currentID] {
			if !visited[nextID] {
				newPath := make([]worldID, len(path)+1)
				copy(newPath, path)
				newPath[len(path)] = nextID
				queue = append(queue, newPath)
			}
		}
	}

	return violations
}

func (w world) label() string {
	strs := make([]string, 0)
	strs = append(strs, "StateMachines:")
	smIDs := make([]string, 0)
	for _, sm := range w.env.machines {
		smIDs = append(smIDs, sm.id())
	}
	sort.Slice(smIDs, func(i, j int) bool {
		nameI := getStateMachineName(w.env.machines[smIDs[i]])
		nameJ := getStateMachineName(w.env.machines[smIDs[j]])
		if nameI != nameJ {
			return nameI < nameJ
		}
		return smIDs[i] < smIDs[j]
	})
	for _, name := range smIDs {
		sm := w.env.machines[name]
		strs = append(strs, fmt.Sprintf("%s = %s; State: %s", getStateMachineName(sm), getStateMachineDetails(sm), getStateDetails(sm.currentState())))
	}

	strs = append(strs, "\nQueuedEvents:")
	smIDs = make([]string, 0)
	for smID := range w.env.queue {
		smIDs = append(smIDs, smID)
	}
	sort.Strings(smIDs)
	for _, smID := range smIDs {
		for _, e := range w.env.queue[smID] {
			sm := w.env.machines[smID]
			if getEventDetails(e) == noFieldsMessage {
				strs = append(strs, fmt.Sprintf("%s << %s;", getStateMachineName(sm), getEventName(e)))
			} else {
				strs = append(strs, fmt.Sprintf("%s << %s; %s", getStateMachineName(sm), getEventName(e), getEventDetails(e)))
			}
		}
	}
	return strings.Join(strs, "\n")
}

type worldJSON struct {
	InvariantViolation bool               `json:"invariant_violation"`
	StateMachines      []stateMachineJSON `json:"state_machines"`
	QueuedEvents       []eventJSON        `json:"queued_events"`
}

type stateMachineJSON struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	State   string `json:"state"`
	Details string `json:"details"`
}

type eventJSON struct {
	TargetMachine string `json:"target_machine"`
	EventName     string `json:"event_name"`
	Details       string `json:"details"`
}

func (m *model) worldsToJSON() []worldJSON {
	allWorlds := make([]worldJSON, 0, len(m.worlds))
	for _, world := range m.worlds {
		worldJSON := m.worldToJSON(world)
		allWorlds = append(allWorlds, worldJSON)
	}

	sort.Slice(allWorlds, func(i, j int) bool {
		return compareWorlds(allWorlds[i], allWorlds[j])
	})

	return allWorlds
}

func compareWorlds(a, b worldJSON) bool {
	if a.InvariantViolation != b.InvariantViolation {
		return !a.InvariantViolation && b.InvariantViolation
	}

	for i := 0; i < len(a.StateMachines) && i < len(b.StateMachines); i++ {
		if a.StateMachines[i].ID != b.StateMachines[i].ID {
			return a.StateMachines[i].ID < b.StateMachines[i].ID
		}
		if a.StateMachines[i].State != b.StateMachines[i].State {
			return a.StateMachines[i].State < b.StateMachines[i].State
		}
		if a.StateMachines[i].Details != b.StateMachines[i].Details {
			return a.StateMachines[i].Details < b.StateMachines[i].Details
		}
	}
	if len(a.StateMachines) != len(b.StateMachines) {
		return len(a.StateMachines) < len(b.StateMachines)
	}

	for i := 0; i < len(a.QueuedEvents) && i < len(b.QueuedEvents); i++ {
		if a.QueuedEvents[i].TargetMachine != b.QueuedEvents[i].TargetMachine {
			return a.QueuedEvents[i].TargetMachine < b.QueuedEvents[i].TargetMachine
		}
		if a.QueuedEvents[i].EventName != b.QueuedEvents[i].EventName {
			return a.QueuedEvents[i].EventName < b.QueuedEvents[i].EventName
		}
		if a.QueuedEvents[i].Details != b.QueuedEvents[i].Details {
			return a.QueuedEvents[i].Details < b.QueuedEvents[i].Details
		}
	}
	return len(a.QueuedEvents) < len(b.QueuedEvents)
}

func (*model) worldToJSON(w world) worldJSON {
	smIDs := make([]string, 0, len(w.env.machines))
	for smID := range w.env.machines {
		smIDs = append(smIDs, smID)
	}
	sort.Strings(smIDs)

	stateMachines := make([]stateMachineJSON, 0, len(smIDs))
	for _, smID := range smIDs {
		sm := w.env.machines[smID]
		stateMachines = append(stateMachines, stateMachineJSON{
			ID:      smID,
			Name:    getStateMachineName(sm),
			State:   getStateDetails(sm.currentState()),
			Details: getStateMachineDetails(sm),
		})
	}

	queuedEvents := make([]eventJSON, 0)
	for _, smID := range smIDs {
		if events, ok := w.env.queue[smID]; ok {
			for _, event := range events {
				queuedEvents = append(queuedEvents, eventJSON{
					TargetMachine: getStateMachineName(w.env.machines[smID]),
					EventName:     getEventName(event),
					Details:       getEventDetails(event),
				})
			}
		}
	}

	sort.Slice(queuedEvents, func(i, j int) bool {
		if queuedEvents[i].TargetMachine != queuedEvents[j].TargetMachine {
			return queuedEvents[i].TargetMachine < queuedEvents[j].TargetMachine
		}
		if queuedEvents[i].EventName != queuedEvents[j].EventName {
			return queuedEvents[i].EventName < queuedEvents[j].EventName
		}
		return queuedEvents[i].Details < queuedEvents[j].Details
	})

	return worldJSON{
		InvariantViolation: len(w.failedInvariants) > 0,
		StateMachines:      stateMachines,
		QueuedEvents:       queuedEvents,
	}
}

func (m *model) summarize(executionTimeMs int64) *modelSummary {
	summary := &modelSummary{
		TotalWorlds:     len(m.worlds),
		ExecutionTimeMs: executionTimeMs,
	}
	return summary
}

var (
	abstractStateMachineType = reflect.TypeOf((*AbstractStateMachine)(nil)).Elem()
	abstractEventType        = reflect.TypeOf((*AbstractEvent)(nil)).Elem()
	abstractStateType        = reflect.TypeOf((*AbstractState)(nil)).Elem()
)

func isGoatInterface(t reflect.Type) bool {
	return t.Implements(abstractStateMachineType) ||
		t.Implements(abstractEventType) ||
		t.Implements(abstractStateType)
}

func warnShallowPointerFields(w io.Writer, sms []AbstractStateMachine) {
	checked := make(map[reflect.Type]bool)
	var checkType func(t reflect.Type)
	checkType = func(t reflect.Type) {
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if checked[t] {
			return
		}
		checked[t] = true

		if t.Kind() != reflect.Struct {
			return
		}

		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			switch f.Type.Kind() {
			case reflect.Ptr:
				if isGoatInterface(f.Type) || isGoatInterface(f.Type.Elem()) {
					continue
				}
				_, _ = fmt.Fprintf(w, "WARNING: type %q has pointer field %q (%s) which will be shared between states during model checking, potentially causing incorrect results. Consider using a value type instead.\n",
					t.Name(), f.Name, f.Type)
			case reflect.Struct:
				checkType(f.Type)
			}
		}
	}

	for _, sm := range sms {
		smType := reflect.TypeOf(sm)
		checkType(smType)

		innerSM := getInnerStateMachine(sm)
		for state, builders := range innerSM.HandlerBuilders {
			stateType := reflect.TypeOf(state)
			checkType(stateType)

			for _, bi := range builders {
				eventType := reflect.TypeOf(bi.event)
				checkType(eventType)
			}
		}
	}
}
