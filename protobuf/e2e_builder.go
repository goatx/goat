package protobuf

import (
	"fmt"
	"reflect"

	"github.com/goatx/goat"
	"github.com/goatx/goat/internal/e2egen"
	"github.com/goatx/goat/internal/strcase"
	"github.com/goatx/goat/internal/typeutil"
)

func buildTestSuite(opts E2ETestOptions) (e2egen.TestSuite, error) {
	suite := e2egen.TestSuite{
		TestPackage: opts.ServiceSchemaPackage,
	}

	for _, svc := range opts.Services {
		serviceName := svc.Spec.getServiceName()
		if svc.TestPackage == "" {
			return e2egen.TestSuite{}, fmt.Errorf("service package is required")
		}

		methods := make([]e2egen.Operation, 0, len(svc.Methods))

		for _, method := range svc.Methods {
			cases := make([]e2egen.TestCase, 0, len(method.TestInputs))

			for ii, input := range method.TestInputs {
				output, err := executeHandler(svc.Spec, method.MethodName, input)
				if err != nil {
					return e2egen.TestSuite{}, fmt.Errorf("service (%s) method (%s) input (%d): failed to execute handler: %w",
						serviceName, method.MethodName, ii, err)
				}

				inputData := serializeMessage(input)
				outputData := serializeMessage(output)

				cases = append(cases, e2egen.TestCase{
					Name:       fmt.Sprintf("case_%d", ii),
					InputType:  typeutil.Name(input),
					Input:      inputData,
					OutputType: typeutil.Name(output),
					Output:     outputData,
				})
			}

			methods = append(methods, e2egen.Operation{
				Name:      method.MethodName,
				TestCases: cases,
			})
		}

		suite.Groups = append(suite.Groups, e2egen.TestGroup{
			Name:          serviceName,
			SchemaPackage: svc.TestPackage,
			Operations:    methods,
		})
	}

	return suite, nil
}

func executeHandler(spec AbstractServiceSpec, methodName string, input AbstractMessage) (AbstractMessage, error) {
	handlers := spec.getHandlers()
	handler, ok := handlers[methodName]
	if !ok {
		return nil, fmt.Errorf("no handler found for method %s", methodName)
	}

	instance, err := spec.newInstance()
	if err != nil {
		return nil, fmt.Errorf("failed to create state machine instance: %w", err)
	}

	ctx := goat.NewHandlerContext(instance)
	return handler(ctx, input, instance), nil
}

func serializeMessage(msg AbstractMessage) map[string]any {
	data := make(map[string]any)

	val := reflect.ValueOf(msg)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		if !field.IsExported() || field.Anonymous || field.Name == "_" {
			continue
		}
		if typeutil.IsEventType(field.Type) || isMessageType(field.Type) {
			continue
		}

		data[strcase.ToProtobufFieldName(field.Name)] = fieldVal.Interface()
	}

	return data
}
