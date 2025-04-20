package main

import (
	"context"
	"log"

	"github.com/goatx/goat"
)

// User Defined Types for goat

type (
	// Context is a user-defined type for the context of the event.
	Context struct {
		RequestID string
	}
)

// Event Definitions

type (
	// eCheckMenuExistenceRequest is an event for checking the existence of the menu.
	// This event is sent from the client to the server.
	eCheckMenuExistenceRequest struct {
		goat.Event // [MUST] Embed the Event struct.
		Ctx        Context
		MenuID     string
		From       *Client
	}
	// eCheckMenuExistenceResponse is an event for the response of checking the existence of the menu.
	// This event is sent from the server to the client.
	eCheckMenuExistenceResponse struct {
		goat.Event // [MUST] Embed the Event struct.
		Exists     bool
		Err        bool
	}
)

// StateMachine State Definitions

type (
	// ClientStateType represents possible states of a Client.
	ClientStateType string
	// ServerStateType represents possible states of a Server.
	ServerStateType string
)

// Client states
const (
	ClientStateIdle    ClientStateType = "idle"    // The client is idle
	ClientStateWaiting ClientStateType = "waiting" // The client is waiting for a response
)

// Server states
const (
	ServerStateInit    ServerStateType = "init"    // The server is initializing
	ServerStateRunning ServerStateType = "running" // The server is running
)

type (
	// ClientState represents the state of a Client.
	ClientState struct {
		goat.State                  // [MUST] Embed the State struct.
		ClientState ClientStateType // The current state of the client
	}

	// ServerState represents the state of a Server.
	ServerState struct {
		goat.State                  // [MUST] Embed the State struct.
		ServerState ServerStateType // The current state of the server
	}
)

// StateMachine Definition

type (
	Client struct {
		goat.StateMachine // [MUST] Embed the StateMachine struct.
		// server is the server that the client is connected to
		Server *Server
	}

	Server struct {
		goat.StateMachine // [MUST] Embed the StateMachine struct.
	}
)

func createClientServerModel() []goat.Option {
	// === Server Spec ===
	serverSpec := goat.NewStateMachineSpec(&Server{})
	serverInit := &ServerState{ServerState: ServerStateInit}
	serverRunning := &ServerState{ServerState: ServerStateRunning}

	serverSpec.
		DefineStates(serverInit, serverRunning).
		SetInitialState(serverInit)

	goat.OnEntry(serverSpec, serverInit, func(ctx context.Context, server *Server) {
		goat.Goto(ctx, serverRunning)
	})

	goat.OnEvent(serverSpec, serverRunning, &eCheckMenuExistenceRequest{},
		func(ctx context.Context, event *eCheckMenuExistenceRequest, server *Server) {
			goat.SendTo(ctx, event.From, &eCheckMenuExistenceResponse{
				Exists: true,
			})
		},
	)

	goat.OnEvent(serverSpec, serverRunning, &eCheckMenuExistenceRequest{},
		func(ctx context.Context, event *eCheckMenuExistenceRequest, server *Server) {
			goat.SendTo(ctx, event.From, &eCheckMenuExistenceResponse{
				Exists: false,
				Err:    false,
			})
		},
	)

	goat.OnEvent(serverSpec, serverRunning, &eCheckMenuExistenceRequest{},
		func(ctx context.Context, event *eCheckMenuExistenceRequest, server *Server) {
			goat.SendTo(ctx, event.From, &eCheckMenuExistenceResponse{
				Exists: false,
				Err:    true,
			})
		},
	)

	// === Client Spec ===
	clientSpec := goat.NewStateMachineSpec(&Client{})
	clientIdle := &ClientState{ClientState: ClientStateIdle}
	clientWaiting := &ClientState{ClientState: ClientStateWaiting}

	clientSpec.
		DefineStates(clientIdle, clientWaiting).
		SetInitialState(clientIdle)

	goat.OnEntry(clientSpec, clientIdle, func(ctx context.Context, client *Client) {
		reqCtx := Context{RequestID: randomRequestID()}
		goat.SendTo(ctx, client.Server, &eCheckMenuExistenceRequest{
			From:   client,
			Ctx:    reqCtx,
			MenuID: "menu_id",
		})
		goat.Goto(ctx, clientWaiting)
	})

	goat.OnEvent(clientSpec, clientWaiting, &eCheckMenuExistenceResponse{},
		func(ctx context.Context, event *eCheckMenuExistenceResponse, client *Client) {
			if event.Err {
				goat.Goto(ctx, clientIdle)
				return
			}
		},
	)

	// === Create Instances ===
	server, err := serverSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	client, err := clientSpec.NewInstance()
	if err != nil {
		log.Fatal(err)
	}
	client.Server = server

	opts := []goat.Option{
		goat.WithStateMachines(server, client),
	}

	return opts
}

func main() {
	opts := createClientServerModel()
	err := goat.Test(opts...)
	if err != nil {
		panic(err)
	}
}

func randomRequestID() string {
	return "random_request_id"
}
