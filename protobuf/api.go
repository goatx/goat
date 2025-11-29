package protobuf

import (
	"context"
	"fmt"
	"reflect"

	"github.com/goatx/goat"
)

// ProtobufResponse represents a response that will be sent back in a Protobuf RPC handler.
// You must create this by calling ProtobufSendTo with your response message.
type ProtobufResponse[O AbstractProtobufMessage] interface {
	protobufResponse()
}

type protobufResponseImpl[O AbstractProtobufMessage] struct {
	event O
}

func (r protobufResponseImpl[O]) protobufResponse() {}

func ProtobufSendTo[O AbstractProtobufMessage](ctx context.Context, target goat.AbstractStateMachine, event O) ProtobufResponse[O] {
	goat.SendTo(ctx, target, event)
	return protobufResponseImpl[O]{event: event}
}

func NewProtobufServiceSpec[T goat.AbstractStateMachine](prototype T) *ProtobufServiceSpec[T] {
	return &ProtobufServiceSpec[T]{
		StateMachineSpec: goat.NewStateMachineSpec(prototype),
		rpcMethods:       []rpcMethod{},
		messages:         make(map[string]*protoMessage),
	}
}

func OnProtobufMessage[T goat.AbstractStateMachine, I AbstractProtobufMessage, O AbstractProtobufMessage](
	spec *ProtobufServiceSpec[T],
	state goat.AbstractState,
	methodName string,
	handler func(context.Context, I, T) ProtobufResponse[O],
) {
	inputEvent := newProtobufMessagePrototype[I]()
	outputEvent := newProtobufMessagePrototype[O]()

	serviceTypeName := getServiceTypeName(spec.StateMachineSpec)
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

	wrappedHandler := func(ctx context.Context, event I, sm T) {
		response := handler(ctx, event, sm)
		_ = response
	}

	goat.OnEvent(spec.StateMachineSpec, state, wrappedHandler)
}

func newProtobufMessagePrototype[T AbstractProtobufMessage]() T {
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
			panic(fmt.Sprintf("type %s does not implement AbstractProtobufMessage", msgType))
		}
		return msg
	}

	value := reflect.New(msgType).Elem().Interface()
	msg, ok := value.(T)
	if !ok {
		panic(fmt.Sprintf("type %s does not implement AbstractProtobufMessage", msgType))
	}
	return msg
}

func analyzeMessage[M AbstractProtobufMessage](instance M) *protoMessage {
	msgType := reflect.TypeOf(instance)
	if msgType.Kind() == reflect.Ptr {
		msgType = msgType.Elem()
	}

	var fields []protoField
	fieldNum := 1

	for i := 0; i < msgType.NumField(); i++ {
		field := msgType.Field(i)

		if !field.IsExported() {
			continue
		}
		if field.Type == reflect.TypeOf(ProtobufMessage[goat.AbstractStateMachine, goat.AbstractStateMachine]{}) {
			continue
		}
		if isGoatEventType(field.Type) {
			continue
		}
		if field.Name == "_" {
			continue
		}

		protoType, isRepeated := mapGoFieldToProto(field.Type)
		if protoType == "" {
			continue
		}

		fields = append(fields, protoField{
			Name:       field.Name,
			Type:       protoType,
			Number:     fieldNum,
			IsRepeated: isRepeated,
		})
		fieldNum++
	}

	return &protoMessage{
		Name:   msgType.Name(),
		Fields: fields,
	}
}

func isGoatEventType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return t.Kind() == reflect.Struct && t.Name() == "Event" && t.PkgPath() == "github.com/goatx/goat"
}

func mapGoFieldToProto(goType reflect.Type) (string, bool) {
	if goType.Kind() == reflect.Slice {
		elemType, _ := mapGoFieldToProto(goType.Elem())
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

func GenerateProtobuf(opts GenerateOptions, specs ...AbstractProtobufServiceSpec) error {
	if opts.OutputDir == "" {
		opts.OutputDir = "./proto"
	}
	if opts.Filename == "" {
		opts.Filename = "generated.proto"
	}

	generator := newProtobufGenerator(opts)
	return generator.generateFromSpecs(specs...)
}

func getServiceTypeName[T goat.AbstractStateMachine](_ *goat.StateMachineSpec[T]) string {
	var zero T
	return getTypeName(zero)
}

func getEventTypeName[E AbstractProtobufMessage](event E) string {
	return getTypeName(event)
}

func getTypeName(v any) string {
	t := reflect.TypeOf(v)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}
