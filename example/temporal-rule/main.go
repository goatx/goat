package main

import (
	"context"
	"log"

	"github.com/goatx/goat"
)

type (
	StateType string
)

const (
	StatePending StateType = "Pending"
	StatePaid    StateType = "Paid"
	StateShipped StateType = "Shipped"
)

type (
	eShipRequest struct {
		goat.Event[*Order, *Shipper]
	}
	eShipResponse struct {
		goat.Event[*Shipper, *Order]
	}
)

type (
	State struct {
		goat.State
		StateType StateType
	}
	Order struct {
		goat.StateMachine
		Shipper *Shipper
	}
)

type (
	shipperState struct{ goat.State }
	Shipper      struct{ goat.StateMachine }
)

func createTemporalRuleModel() []goat.Option {
	// === Shipper Spec ===
	shipperSpec := goat.NewStateMachineSpec(&Shipper{})
	idle := &shipperState{}
	shipperSpec.DefineStates(idle).SetInitialState(idle)
	goat.OnEvent(shipperSpec, idle, func(ctx context.Context, e *eShipRequest, _ *Shipper) {
		goat.SendTo(ctx, e.Sender(), &eShipResponse{})
	})

	// === Order Spec ===
	orderSpec := goat.NewStateMachineSpec(&Order{})
	pending := &State{StateType: StatePending}
	paid := &State{StateType: StatePaid}
	shipped := &State{StateType: StateShipped}

	orderSpec.DefineStates(pending, paid, shipped).SetInitialState(pending)

	goat.OnEntry(orderSpec, pending, func(ctx context.Context, _ *Order) {
		goat.Goto(ctx, paid)
	})
	goat.OnEntry(orderSpec, paid, func(ctx context.Context, o *Order) {
		goat.SendTo(ctx, o.Shipper, &eShipRequest{})
	})
	goat.OnEvent(orderSpec, paid, func(ctx context.Context, _ *eShipResponse, o *Order) {
		goat.Goto(ctx, shipped)
	})

	// === Create Instances ===
	shipper, err := shipperSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	order, err := orderSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	order.Shipper = shipper

	inPaid := goat.NewCondition("inPaid", order, func(o *Order) bool {
		return o.State.(*State).StateType == StatePaid
	})
	inShipped := goat.NewCondition("inShipped", order, func(o *Order) bool {
		return o.State.(*State).StateType == StateShipped
	})

	rule := goat.WheneverPEventuallyQ(inPaid, inShipped)

	opts := []goat.Option{
		goat.WithStateMachines(shipper, order),
		goat.WithRules(rule),
	}

	return opts
}

func main() {
	opts := createTemporalRuleModel()
	_, err := goat.Test(opts...)
	if err != nil {
		panic(err)
	}
}
