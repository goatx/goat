package protobuf

import (
	"context"
	"reflect"
	"strings"

	"github.com/goatx/goat"
	"github.com/goatx/goat/internal/typeutil"
)

type handlerFunc func(ctx context.Context, input AbstractMessage, sm goat.AbstractStateMachine) AbstractMessage

type AbstractMessage interface {
	isMessage() bool
	goat.AbstractEvent
}

type Message[Sender goat.AbstractStateMachine, Recipient goat.AbstractStateMachine] struct {
	goat.Event[Sender, Recipient]
	// this is needed to make Message copyable
	_ rune
}

func (*Message[Sender, Recipient]) isMessage() bool {
	return true
}

func isMessageType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct || t.PkgPath() != "github.com/goatx/goat/protobuf" {
		return false
	}

	name := t.Name()
	return strings.HasPrefix(name, "Message[")
}

type AbstractServiceSpec interface {
	isServiceSpec() bool
	getRPCMethods() []rpcMethod
	getMessages() map[string]*message
	getHandlers() map[string]handlerFunc
	getServiceName() string
	newInstance() (goat.AbstractStateMachine, error)
}

type ServiceSpec[T goat.AbstractStateMachine] struct {
	*goat.StateMachineSpec[T]
	rpcMethods []rpcMethod
	messages   map[string]*message
	handlers   map[string]handlerFunc
}

func (*ServiceSpec[T]) isServiceSpec() bool {
	return true
}

func (ps *ServiceSpec[T]) getRPCMethods() []rpcMethod {
	return ps.rpcMethods
}

func (ps *ServiceSpec[T]) getMessages() map[string]*message {
	return ps.messages
}

func (ps *ServiceSpec[T]) getHandlers() map[string]handlerFunc {
	return ps.handlers
}

func (*ServiceSpec[T]) getServiceName() string {
	return typeutil.NameOf[T]()
}

func (ps *ServiceSpec[T]) newInstance() (goat.AbstractStateMachine, error) {
	return ps.NewInstance()
}

func (ps *ServiceSpec[T]) addRPCMethod(metadata rpcMethod) {
	ps.rpcMethods = append(ps.rpcMethods, metadata)
}

func (ps *ServiceSpec[T]) addMessage(msg *message) {
	if ps.messages == nil {
		ps.messages = make(map[string]*message)
	}
	ps.messages[msg.Name] = msg
}

type rpcMethod struct {
	ServiceType string
	MethodName  string
	InputType   string
	OutputType  string
}

type message struct {
	Name   string
	Fields []field
}

type field struct {
	Name       string
	Type       string
	Number     int
	IsRepeated bool
}
