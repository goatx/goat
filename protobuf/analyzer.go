package protobuf

import (
	"reflect"
	"sort"
)

type typeAnalyzer struct {
	processedTypes map[reflect.Type]bool
}

func newTypeAnalyzer() *typeAnalyzer {
	return &typeAnalyzer{
		processedTypes: make(map[reflect.Type]bool),
	}
}

type service struct {
	Name    string
	Methods []method
}

type method struct {
	Name       string
	InputType  string
	OutputType string
}

type definitions struct {
	Messages []*message
	Services []*service
}

func (a *typeAnalyzer) analyzeSpecs(specs ...AbstractServiceSpec) *definitions {
	definitions := &definitions{
		Messages: []*message{},
		Services: []*service{},
	}

	a.processedTypes = make(map[reflect.Type]bool)

	for _, spec := range specs {
		service := a.analyzeServiceSpecInterface(spec)
		definitions.Services = append(definitions.Services, service)

		messages := make([]*message, 0, len(spec.getMessages()))
		for _, message := range spec.getMessages() {
			messages = append(messages, message)
		}
		sort.Slice(messages, func(i, j int) bool {
			return messages[i].Name < messages[j].Name
		})

		definitions.Messages = append(definitions.Messages, messages...)
	}

	return definitions
}

func (*typeAnalyzer) analyzeServiceSpecInterface(spec AbstractServiceSpec) *service {
	rpcMethods := spec.getRPCMethods()

	serviceName := ""
	if len(rpcMethods) > 0 {
		serviceName = rpcMethods[0].ServiceType
	}

	service := &service{
		Name:    serviceName,
		Methods: []method{},
	}

	for _, metadata := range rpcMethods {
		method := method{
			Name:       metadata.MethodName,
			InputType:  metadata.InputType,
			OutputType: metadata.OutputType,
		}
		service.Methods = append(service.Methods, method)
	}

	return service
}
