package goat

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

type modelSummary struct {
	TotalWorlds int `json:"total_worlds"`

	InvariantViolations struct {
		Found bool `json:"found"`
		Count int  `json:"count"`
	} `json:"invariant_violations"`

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
		sb.WriteString(fmt.Sprintf("%d", id))
		sb.WriteString(` [ label="`)
		sb.WriteString(wld.label())
		sb.WriteString("\" ];\n")
		if id == m.initial.id {
			sb.WriteString("  ")
			sb.WriteString(fmt.Sprintf("%d", id))
			sb.WriteString(" [ penwidth=5 ];\n")
		}
		if wld.invariantViolation {
			sb.WriteString("  ")
			sb.WriteString(fmt.Sprintf("%d", id))
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
			sb.WriteString(fmt.Sprintf("%d", to))
			sb.WriteString(";\n")
		}
	}
	sb.WriteString("}\n")

	// ------------ Output ------------
	_, _ = io.WriteString(w, sb.String())
}

func (m *model) writeLog(w io.Writer, invariantDescription string) {
	var sb strings.Builder
	paths := m.findPathsToViolations()

	if len(paths) == 0 {
		sb.WriteString("No invariant violations found.\n")
		_, _ = io.WriteString(w, sb.String())
		return
	}

	for i, path := range paths {
		if i > 0 {
			sb.WriteString("\n")
		}

		sb.WriteString("InvariantError:  ")
		sb.WriteString(invariantDescription)
		sb.WriteString("   ✘\n")
		sb.WriteString("Path (length = ")
		sb.WriteString(fmt.Sprintf("%d", len(path)))
		sb.WriteString("):\n")

		for j, worldID := range path {
			world := m.worlds[worldID]

			if j == len(path)-1 && world.invariantViolation {
				sb.WriteString("  [")
				sb.WriteString(fmt.Sprintf("%d", j))
				sb.WriteString("] <-- violation here\n")
			} else {
				sb.WriteString("  [")
				sb.WriteString(fmt.Sprintf("%d", j))
				sb.WriteString("]\n")
			}
			sb.WriteString("  StateMachines:\n")
			for _, sm := range world.env.machines {
				sb.WriteString("    Name: ")
				sb.WriteString(getStateMachineName(sm))
				sb.WriteString(", Detail: ")
				sb.WriteString(getStateMachineDetails(sm))
				sb.WriteString(", State: ")
				sb.WriteString(getStateDetails(sm.currentState()))
				sb.WriteString("\n")
			}
			sb.WriteString("  QueuedEvents:\n")
			for smID, events := range world.env.queue {
				for _, event := range events {
					sb.WriteString("    StateMachine: ")
					sb.WriteString(getStateMachineName(world.env.machines[smID]))
					sb.WriteString(", Event: ")
					sb.WriteString(getEventName(event))
					sb.WriteString(", Detail: ")
					sb.WriteString(getEventDetails(event))
					sb.WriteString("\n")
				}
			}
		}
	}

	_, _ = io.WriteString(w, sb.String())
}

func (m *model) findPathsToViolations() [][]worldID {
	var paths [][]worldID

	visited := make(map[worldID]bool)

	queue := [][]worldID{{m.initial.id}}

	for len(queue) > 0 {
		path := queue[0]
		queue = queue[1:]

		currentID := path[len(path)-1]

		if visited[currentID] {
			continue
		}
		visited[currentID] = true

		if m.worlds[currentID].invariantViolation {
			paths = append(paths, path)
			continue
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

	return paths
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
		InvariantViolation: w.invariantViolation,
		StateMachines:      stateMachines,
		QueuedEvents:       queuedEvents,
	}
}

func (m *model) summarize(executionTimeMs int64) *modelSummary {
	summary := &modelSummary{
		TotalWorlds:     len(m.worlds),
		ExecutionTimeMs: executionTimeMs,
	}

	violationCount := 0
	for _, world := range m.worlds {
		if world.invariantViolation {
			violationCount++
		}
	}

	summary.InvariantViolations.Found = violationCount > 0
	summary.InvariantViolations.Count = violationCount

	return summary
}
