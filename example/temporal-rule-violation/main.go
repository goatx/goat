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
		goat.Event[*Order, *FailingShipper]
	}
	eShipResponse struct {
		goat.Event[*FailingShipper, *Order]
	}
)

type (
	State struct {
		goat.State
		StateType StateType
	}
	Order struct {
		goat.StateMachine
		Shipper *FailingShipper
	}
)

type (
	shipperState   struct{ goat.State }
	FailingShipper struct{ goat.StateMachine }
)

func createTemporalRuleViolationModel() []goat.Option {
	// === Shipper Spec ===
	shipperSpec := goat.NewStateMachineSpec(&FailingShipper{})
	idle := &shipperState{}
	shipperSpec.DefineStates(idle).SetInitialState(idle)
	goat.OnEvent(shipperSpec, idle, func(ctx context.Context, _ *eShipRequest, _ *FailingShipper) {
		// Intentionally left blank: the failing shipper never sends the response.
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
	opts := createTemporalRuleViolationModel()
	_, err := goat.Test(opts...)
	if err != nil {
		panic(err)
	}
}
