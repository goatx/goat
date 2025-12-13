package protobuf

import (
	"context"
	"fmt"
	"reflect"

	"github.com/goatx/goat"
	"github.com/goatx/goat/internal/typeutil"
)

type Response[O AbstractMessage] struct {
	event O
}

func (r Response[O]) getEvent() AbstractMessage {
	return r.event
}

func SendTo[O AbstractMessage](ctx context.Context, target goat.AbstractStateMachine, event O) Response[O] {
	goat.SendTo(ctx, target, event)
	return Response[O]{event: event}
}

func NewServiceSpec[T goat.AbstractStateMachine](prototype T) *ServiceSpec[T] {
	return &ServiceSpec[T]{
		StateMachineSpec: goat.NewStateMachineSpec(prototype),
		rpcMethods:       []rpcMethod{},
		messages:         make(map[string]*message),
		handlers:         make(map[string]handlerFunc),
	}
}

func OnMessage[T goat.AbstractStateMachine, I AbstractMessage, O AbstractMessage](
	spec *ServiceSpec[T],
	state goat.AbstractState,
	methodName string,
	handler func(context.Context, I, T) Response[O],
) {
	inputEvent := newMessagePrototype[I]()
	outputEvent := newMessagePrototype[O]()

	serviceTypeName := spec.getServiceName()
	inputTypeName := getEventTypeName(inputEvent)
	outputTypeName := getEventTypeName(outputEvent)

	metadata := rpcMethod{
		ServiceType: serviceTypeName,
		MethodName:  methodName,
		InputType:   inputTypeName,
		OutputType:  outputTypeName,
	}

	spec.addRPCMethod(metadata)

	inputMsg := analyzeMessage(inputEvent)
	outputMsg := analyzeMessage(outputEvent)
	spec.addMessage(inputMsg)
	spec.addMessage(outputMsg)

	spec.handlers[methodName] = func(ctx context.Context, input AbstractMessage, sm goat.AbstractStateMachine) AbstractMessage {
		return handler(ctx, input.(I), sm.(T)).getEvent()
	}

	wrappedHandler := func(ctx context.Context, event I, sm T) {
		response := handler(ctx, event, sm)
		_ = response
	}

	goat.OnEvent(spec.StateMachineSpec, state, wrappedHandler)
}

func newMessagePrototype[T AbstractMessage]() T {
	var zero T
	msgType := reflect.TypeOf(zero)
	if msgType == nil {
		msgType = reflect.TypeFor[T]()
	}

	if msgType.Kind() == reflect.Interface {
		panic(fmt.Sprintf("cannot use interface type %s as protobuf message type parameter; use a concrete message type instead", msgType))
	}

	if msgType.Kind() == reflect.Pointer {
		elem := msgType.Elem()
		if elem.Kind() == reflect.Interface {
			panic(fmt.Sprintf("cannot use interface type %s as protobuf message type parameter; use a concrete message type instead", elem))
		}

		prototype := reflect.New(elem).Interface()
		msg, ok := prototype.(T)
		if !ok {
			panic(fmt.Sprintf("type %s does not implement AbstractMessage", msgType))
		}
		return msg
	}

	value := reflect.New(msgType).Elem().Interface()
	msg, ok := value.(T)
	if !ok {
		panic(fmt.Sprintf("type %s does not implement AbstractMessage", msgType))
	}
	return msg
}

func analyzeMessage[M AbstractMessage](instance M) *message {
	msgType := reflect.TypeOf(instance)
	if msgType.Kind() == reflect.Ptr {
		msgType = msgType.Elem()
	}

	var fields []field
	fieldNum := 1

	for i := 0; i < msgType.NumField(); i++ {
		f := msgType.Field(i)

		if !f.IsExported() || f.Name == "_" {
			continue
		}
		if typeutil.IsEventType(f.Type) || isMessageType(f.Type) {
			continue
		}

		protoType, isRepeated := mapGoField(f.Type)
		if protoType == "" {
			continue
		}

		fields = append(fields, field{
			Name:       f.Name,
			Type:       protoType,
			Number:     fieldNum,
			IsRepeated: isRepeated,
		})
		fieldNum++
	}

	return &message{
		Name:   msgType.Name(),
		Fields: fields,
	}
}

func mapGoField(goType reflect.Type) (string, bool) {
	if goType.Kind() == reflect.Slice {
		elemType, _ := mapGoField(goType.Elem())
		return elemType, true
	}

	switch goType.Kind() {
	case reflect.String:
		return "string", false
	case reflect.Bool:
		return "bool", false
	case reflect.Int32:
		return "int32", false
	case reflect.Int64, reflect.Int:
		return "int64", false
	case reflect.Float32:
		return "float", false
	case reflect.Float64:
		return "double", false
	default:
		return "", false
	}
}

type GenerateOptions struct {
	OutputDir   string
	PackageName string
	GoPackage   string
	Filename    string
}

func Generate(opts GenerateOptions, specs ...AbstractServiceSpec) error {
	if opts.OutputDir == "" {
		opts.OutputDir = "./proto"
	}
	if opts.Filename == "" {
		opts.Filename = "generated.proto"
	}

	generator := newGenerator(opts)
	return generator.generateFromSpecs(specs...)
}

func getEventTypeName[E AbstractMessage](event E) string {
	return typeutil.Name(event)
}
