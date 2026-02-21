# goat

## What is goat

goat is a **Design as Code** library for Go. You write software design — components, states, behaviors, and interactions — as Go code, not documents.

A goat specification is executable. goat can **model check** it by exploring every possible execution order and verifying your rules against every reachable state. It can also **generate schemas** (Protocol Buffers, OpenAPI) from the same specification.

Design written as code doesn't drift from implementation. It lives in your repository and you can verify it by running it.

## How goat Models Systems

goat models a system as a composition of **state machines** that communicate through **events**.

Every component — a server, a client, a database, a message broker — becomes a state machine with discrete states. A server might be _initializing_ or _running_. By adding states like _crashed_ or _unavailable_, you can verify how the rest of the system behaves under failure.

State machines communicate by sending **events** through queues. Events are asynchronous — the sender doesn't wait for the recipient to process the event. This directly models how distributed systems communicate through HTTP requests, message queues, gRPC calls, and database queries.

**Handlers** define what a state machine does when it enters a state or receives an event: update its data, transition to another state, or send events to other machines.

Real systems are non-deterministic. For example, a request might return a successful response or an error. goat models this with **multiple handlers** for the same state and event, one per possible outcome. goat explores every path across all machines.

## What goat Checks

goat verifies two kinds of properties:

**Safety — "this bad thing never happens."** For example, an account balance never goes below zero. In goat, you write this as `Always(condition)`: the condition must hold in every reachable state. If there is a violation, goat can show it as a finite sequence of steps leading to the bad state.

**Liveness — "this good thing eventually happens."** For example, whenever an order is paid, it is eventually shipped. goat provides temporal rules for this:

- `WheneverPEventuallyQ(p, q)` — whenever p becomes true, q must eventually become true.
- `EventuallyAlways(c)` — the system eventually reaches a state where c holds permanently.
- `AlwaysEventually(c)` — c keeps becoming true repeatedly, forever.

When goat finds a violation, it reports the exact sequence of steps — which machine processed which event in which order — so you can see the concrete scenario your system can hit.

## Writing Specifications

A goat specification is written in three steps:

1. **Define state machine and event schemas** — What components exist, what states they can be in, and what messages they exchange.
2. **Define rules** — What properties the system must satisfy.
3. **Define handlers** — How each component behaves. What it does when it enters a state or receives an event.

### Defining State Machine and Event Schemas

A state is a struct that embeds `goat.State`. Fields distinguish one state from another:

```go
type MyState struct {
    goat.State
    Status string
}

var (
    stateA = &MyState{Status: "A"}
    stateB = &MyState{Status: "B"}
)
```

A state machine is a struct that embeds `goat.StateMachine`. Add fields for any domain data the component carries:

```go
type Server struct {
    goat.StateMachine
    ConnectionCount int
}
```

An event is a struct that embeds `goat.Event[Sender, Recipient]`. The type parameters specify which state machine sends and which receives:

```go
type Request struct {
    goat.Event[*Client, *Server]
    Payload string
}
```

Create a specification, define the possible states, and set the initial state:

```go
spec := goat.NewStateMachineSpec(&Server{})
spec.DefineStates(stateA, stateB).SetInitialState(stateA)
```

Create instances from a spec. Each instance is an independent state machine. Every unit of work that can run concurrently needs its own instance — if a server handles two requests in parallel, that is two server instances, even on a single machine:

```go
db, _ := dbSpec.NewInstance()
server1, _ := serverSpec.NewInstance()
server2, _ := serverSpec.NewInstance()
client1, _ := clientSpec.NewInstance()
client2, _ := clientSpec.NewInstance()
```

The model checker explores every interleaving of events across all instances.

To send an event to another machine with `SendTo`, the sender needs a reference to the target. Store it as a field and assign it after creating instances:

```go
type Client struct {
    goat.StateMachine
    Server *Server
}

client, _ := clientSpec.NewInstance()
server, _ := serverSpec.NewInstance()
client.Server = server
```

### Defining Rules

A **condition** is a named boolean check on one or more state machines. A **rule** takes a condition and specifies how to verify it — always true, eventually true, and so on.

#### Conditions on a single machine

`NewCondition` creates a condition that inspects one state machine. The check function receives the typed machine instance:

```go
cond := goat.NewCondition("non-negative", sm, func(sm *MyMachine) bool {
    return sm.Count >= 0
})
```

#### Conditions on multiple machines

When a property involves more than one machine, use `NewCondition2` or `NewCondition3`:

```go
goat.NewCondition2("a-b-consistent", smA, smB, func(a *MachineA, b *MachineB) bool {
    return a.Value == b.Value
})

goat.NewCondition3("triple", smA, smB, smC, func(a *MachineA, b *MachineB, c *MachineC) bool {
    return a.Total == b.Total+c.Total
})
```

For four or more machines, use `NewMultiCondition`. The check function receives a `Machines` accessor, and you retrieve each machine with `GetMachine`:

```go
goat.NewMultiCondition("all-ready", func(machines goat.Machines) bool {
    a, ok := goat.GetMachine(machines, smA)
    if !ok { return false }
    b, ok := goat.GetMachine(machines, smB)
    if !ok { return false }
    c, ok := goat.GetMachine(machines, smC)
    if !ok { return false }
    d, ok := goat.GetMachine(machines, smD)
    if !ok { return false }
    return a.Ready && b.Ready && c.Ready && d.Ready
}, smA, smB, smC, smD)
```

#### Building rules from conditions

Pass conditions to a rule constructor to specify what the model checker verifies.

**`Always`** — the condition must hold in every reachable state:

```go
goat.Always(nonNegative)
```

**`WheneverPEventuallyQ`** — whenever condition p holds, condition q must eventually hold:

```go
goat.WheneverPEventuallyQ(requested, completed)
```

**`EventuallyAlways`** — the condition must eventually become true and stay true permanently:

```go
goat.EventuallyAlways(stable)
```

**`AlwaysEventually`** — the condition must keep becoming true repeatedly, forever:

```go
goat.AlwaysEventually(ready)
```

### Defining Handlers

Handlers define what a state machine does when it enters a state or receives an event.

#### Entry handlers

`OnEntry` registers a handler that runs when a state machine enters a given state:

```go
goat.OnEntry(spec, stateA, func(ctx context.Context, sm *MyMachine) {
    sm.Count = 0
    goat.Goto(ctx, stateB)
})
```

#### Event handlers

`OnEvent` registers a handler that runs when a state machine receives a specific event in a given state:

```go
goat.OnEvent(spec, stateA, func(ctx context.Context, event *MyEvent, sm *MyMachine) {
    sm.Count += event.Amount
    goat.SendTo(ctx, event.Sender(), &MyResponse{Total: sm.Count})
    goat.Goto(ctx, stateB)
})
```

#### Actions inside handlers

Handlers use context functions to perform actions:

- `goat.Goto(ctx, state)` — transition to another state.
- `goat.SendTo(ctx, target, event)` — send an event to another state machine. The target is either a field on the machine (`sm.Server`) or `event.Sender()` to reply to whoever sent the event.
- `goat.Halt(ctx, target)` — stop a state machine permanently.

Every received event exposes `Sender()` and `Recipient()`, typed to the machines declared in `Event[Sender, Recipient]`. Use `event.Sender()` to reply without needing a stored reference.

Handlers can also update the state machine's fields directly, as shown above with `sm.Count`.

#### Non-determinism

Registering multiple handlers for the same state and event models non-determinism. The model checker explores every handler as a separate execution path:

```go
goat.OnEvent(spec, stateA, func(ctx context.Context, event *Request, sm *MyMachine) {
    goat.SendTo(ctx, event.Sender(), &Response{OK: true})
})
goat.OnEvent(spec, stateA, func(ctx context.Context, event *Request, sm *MyMachine) {
    goat.SendTo(ctx, event.Sender(), &Response{OK: false})
})
```

This works with any handler type, not just `OnEvent`. For example, two `OnEntry` handlers for the same state create two possible paths on entry.

#### Other handler types

- `goat.OnExit(spec, state, fn)` — runs when leaving a state.
- `goat.OnTransition(spec, state, fn)` — runs during a transition, after exit and before entry. The callback receives the target state as an additional argument.
- `goat.OnHalt(spec, state, fn)` — runs when the state machine is halted.

### Running Model Checking

`goat.Test` runs the model checker. Pass options to specify which state machines to check and which rules to verify:

```go
err := goat.Test(
    goat.WithStateMachines(server, client),
    goat.WithRules(
        goat.Always(nonNegative),
        goat.WheneverPEventuallyQ(requested, completed),
    ),
)
```

`WithStateMachines` takes the state machine instances to include in the model. `WithRules` takes the rules defined with `Always`, `WheneverPEventuallyQ`, and the other rule constructors.

When a violation is found, `Test` prints the path to the violating state to stdout — which machine was in which state at each step — so you can trace the exact scenario:

```
Condition failed. Not Always non-negative.
Path (length = 3):
  [0]
  StateMachines:
    Name: Server, Detail: {Count: 0}, State: idle
  [1]
  StateMachines:
    Name: Server, Detail: {Count: -1}, State: processing  <-- violation here
```

When no violations are found, `Test` prints a summary with the total number of explored states and execution time.

## Examples

The [`example`](./example) directory contains runnable specifications:

- [`simple-transition`](./example/simple-transition)
  - Basic state transitions
- [`simple-halt`](./example/simple-halt)
  - Halting a state machine
- [`simple-non-deterministic`](./example/simple-non-deterministic)
  - Non-deterministic branching
- [`client-server`](./example/client-server)
  - Event passing between two machines
- [`temporal-rule`](./example/temporal-rule)
  - Temporal rule checking
- [`temporal-rule-violation`](./example/temporal-rule-violation)
  - Detecting a temporal rule violation
- [`meeting-room-reservation/with-exclusion`](./example/meeting-room-reservation/with-exclusion)
  - Meeting room booking with exclusive locking
- [`meeting-room-reservation/without-exclusion`](./example/meeting-room-reservation/without-exclusion)
  - The same scenario without locking, demonstrating a double-booking violation
