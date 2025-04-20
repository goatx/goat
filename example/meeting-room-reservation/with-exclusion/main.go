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
	// Server states
	ServerIdle       StateType = "ServerIdle"
	ServerProcessing StateType = "ServerProcessing"

	// Database states
	DBIdle StateType = "DBIdle"

	// Client states
	ClientIdle       StateType = "ClientIdle"
	ClientRequesting StateType = "ClientRequesting"
	ClientEnd        StateType = "ClientEnd"
)

type State struct {
	goat.State
	StateType StateType
}

type ReservationRequestEvent struct {
	goat.Event
	RoomID   int
	ClientID int
	Client   *ClientStateMachine
}

type ReservationResultEvent struct {
	goat.Event
	RoomID    int
	ClientID  int
	Succeeded bool
}

type DBSelectEvent struct {
	goat.Event
	RoomID   int
	ClientID int
	Server   *ServerStateMachine
}

type DBSelectResultEvent struct {
	goat.Event
	RoomID     int
	ClientID   int
	IsReserved bool
	IsLocked   bool
}

type DBUpdateEvent struct {
	goat.Event
	RoomID   int
	ClientID int
	Server   *ServerStateMachine
}

type DBUpdateResultEvent struct {
	goat.Event
	RoomID    int
	ClientID  int
	Succeeded bool
}

type Reservation struct {
	RoomID   int
	ClientID int
}

// Server state machine
type ServerStateMachine struct {
	goat.StateMachine
	CurrentRequest *ReservationRequestEvent
	DB             *DBStateMachine
}

type DBStateMachine struct {
	goat.StateMachine
	Reservations []Reservation
	LockedRooms  map[int]int
}

type ClientStateMachine struct {
	goat.StateMachine
	ClientID   int
	TargetRoom int
	Server     *ServerStateMachine
}

func createMeetingRoomWithExclusionModel() []goat.Option {
	// === Database Spec ===
	dbSpec := goat.NewStateMachineSpec(&DBStateMachine{})
	dbIdle := &State{StateType: DBIdle}

	dbSpec.DefineStates(dbIdle).SetInitialState(dbIdle)

	goat.OnEvent(dbSpec, dbIdle, &DBSelectEvent{},
		func(ctx context.Context, event *DBSelectEvent, db *DBStateMachine) {
			isReserved := false
			for _, res := range db.Reservations {
				if res.RoomID == event.RoomID {
					isReserved = true
					break
				}
			}

			isLocked := false
			if _, exists := db.LockedRooms[event.RoomID]; !exists {
				db.LockedRooms[event.RoomID] = event.ClientID
				isLocked = true
			}

			resultEvent := &DBSelectResultEvent{
				RoomID:     event.RoomID,
				ClientID:   event.ClientID,
				IsReserved: isReserved,
				IsLocked:   isLocked,
			}

			goat.SendTo(ctx, event.Server, resultEvent)
		})

	goat.OnEvent(dbSpec, dbIdle, &DBUpdateEvent{},
		func(ctx context.Context, event *DBUpdateEvent, db *DBStateMachine) {
			hasLock := false
			if clientID, exists := db.LockedRooms[event.RoomID]; exists && clientID == event.ClientID {
				hasLock = true
			}

			succeeded := false
			if hasLock {
				db.Reservations = append(db.Reservations, Reservation{
					RoomID:   event.RoomID,
					ClientID: event.ClientID,
				})
				succeeded = true
			}

			resultEvent := &DBUpdateResultEvent{
				RoomID:    event.RoomID,
				ClientID:  event.ClientID,
				Succeeded: succeeded,
			}

			goat.SendTo(ctx, event.Server, resultEvent)
		})

	// === Server Spec ===
	serverSpec := goat.NewStateMachineSpec(&ServerStateMachine{})
	serverIdle := &State{StateType: ServerIdle}
	serverProcessing := &State{StateType: ServerProcessing}

	serverSpec.DefineStates(serverIdle, serverProcessing).SetInitialState(serverIdle)

	goat.OnEvent(serverSpec, serverIdle, &ReservationRequestEvent{},
		func(ctx context.Context, event *ReservationRequestEvent, server *ServerStateMachine) {
			server.CurrentRequest = event

			selectEvent := &DBSelectEvent{
				RoomID:   event.RoomID,
				ClientID: event.ClientID,
				Server:   server,
			}

			goat.SendTo(ctx, server.DB, selectEvent)
			goat.Goto(ctx, serverProcessing)
		})

	goat.OnEvent(serverSpec, serverProcessing, &DBSelectResultEvent{},
		func(ctx context.Context, event *DBSelectResultEvent, server *ServerStateMachine) {
			if server.CurrentRequest == nil {
				goat.Goto(ctx, serverIdle)
				return
			}

			if event.IsLocked {
				if !event.IsReserved {
					updateEvent := &DBUpdateEvent{
						RoomID:   server.CurrentRequest.RoomID,
						ClientID: server.CurrentRequest.ClientID,
						Server:   server,
					}

					goat.SendTo(ctx, server.DB, updateEvent)
				} else {
					resultEvent := &ReservationResultEvent{
						RoomID:    server.CurrentRequest.RoomID,
						ClientID:  server.CurrentRequest.ClientID,
						Succeeded: false,
					}

					goat.SendTo(ctx, server.CurrentRequest.Client, resultEvent)
					server.CurrentRequest = nil
					goat.Goto(ctx, serverIdle)
				}
			} else {
				resultEvent := &ReservationResultEvent{
					RoomID:    server.CurrentRequest.RoomID,
					ClientID:  server.CurrentRequest.ClientID,
					Succeeded: false,
				}

				goat.SendTo(ctx, server.CurrentRequest.Client, resultEvent)
				server.CurrentRequest = nil
				goat.Goto(ctx, serverIdle)
			}
		})

	goat.OnEvent(serverSpec, serverProcessing, &DBUpdateResultEvent{},
		func(ctx context.Context, event *DBUpdateResultEvent, server *ServerStateMachine) {
			if server.CurrentRequest == nil {
				goat.Goto(ctx, serverIdle)
				return
			}

			resultEvent := &ReservationResultEvent{
				RoomID:    server.CurrentRequest.RoomID,
				ClientID:  server.CurrentRequest.ClientID,
				Succeeded: event.Succeeded,
			}

			goat.SendTo(ctx, server.CurrentRequest.Client, resultEvent)
			server.CurrentRequest = nil
			goat.Goto(ctx, serverIdle)
		})

	// === Client Spec ===
	clientSpec := goat.NewStateMachineSpec(&ClientStateMachine{})
	clientIdle := &State{StateType: ClientIdle}
	clientRequesting := &State{StateType: ClientRequesting}
	clientEnd := &State{StateType: ClientEnd}

	clientSpec.DefineStates(clientIdle, clientRequesting, clientEnd).SetInitialState(clientIdle)

	goat.OnEntry(clientSpec, clientIdle,
		func(ctx context.Context, client *ClientStateMachine) {
			requestEvent := &ReservationRequestEvent{
				RoomID:   client.TargetRoom,
				ClientID: client.ClientID,
				Client:   client,
			}

			goat.SendTo(ctx, client.Server, requestEvent)
			goat.Goto(ctx, clientRequesting)
		})

	goat.OnEvent(clientSpec, clientRequesting, &ReservationResultEvent{},
		func(ctx context.Context, event *ReservationResultEvent, client *ClientStateMachine) {
			if event.ClientID == client.ClientID {
				if event.Succeeded {
					goat.Goto(ctx, clientEnd)
				} else {
					goat.Goto(ctx, clientEnd)
				}
			}
		})

	goat.OnEntry(clientSpec, clientEnd,
		func(ctx context.Context, client *ClientStateMachine) {
		})

	// === Create Instances ===
	// Create database instance
	db, err := dbSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	db.Reservations = make([]Reservation, 0)
	db.LockedRooms = make(map[int]int)

	// Create server instances
	server1, err := serverSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	server1.DB = db

	server2, err := serverSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	server2.DB = db

	// Create client instances
	client1, err := clientSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	client1.ClientID = 0
	client1.TargetRoom = 101
	client1.Server = server1

	client2, err := clientSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	client2.ClientID = 1
	client2.TargetRoom = 101
	client2.Server = server2

	opts := []goat.Option{
		goat.WithStateMachines(server1, server2, db, client1, client2),
		goat.WithInvariants(
			goat.NewInvariant(db, func(db *DBStateMachine) bool {
				roomClients := make(map[int]map[int]bool)
				for _, res := range db.Reservations {
					if _, ok := roomClients[res.RoomID]; !ok {
						roomClients[res.RoomID] = make(map[int]bool)
					}
					roomClients[res.RoomID][res.ClientID] = true
				}

				for _, clients := range roomClients {
					if len(clients) > 1 {
						return false
					}
				}

				return true
			}),
		),
	}

	return opts
}

func main() {
	opts := createMeetingRoomWithExclusionModel()
	err := goat.Test(opts...)
	if err != nil {
		panic(err)
	}
}
