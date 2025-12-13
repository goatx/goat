package protobuf

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestOnMessage(t *testing.T) {
	tests := []struct {
		name           string
		methodName     string
		expectedMethod rpcMethod
	}{
		{
			name:       "registers single method and verifies goat integration",
			methodName: "TestMethod",
			expectedMethod: rpcMethod{
				ServiceType: "TestService1",
				MethodName:  "TestMethod",
				InputType:   "TestRequest1",
				OutputType:  "TestResponse1",
			},
		},
		{
			name:       "registers method with custom name",
			methodName: "CustomMethod",
			expectedMethod: rpcMethod{
				ServiceType: "TestService1",
				MethodName:  "CustomMethod",
				InputType:   "TestRequest1",
				OutputType:  "TestResponse1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := NewServiceSpec(&TestService1{})
			state := &TestIdleState{}
			spec.DefineStates(state).SetInitialState(state)

			initialMethodCount := len(spec.getRPCMethods())

			OnMessage(spec, state, tt.methodName,
				func(ctx context.Context, event *TestRequest1, sm *TestService1) Response[*TestResponse1] {
					return SendTo(ctx, sm, &TestResponse1{Result: "test"})
				})

			methods := spec.getRPCMethods()
			if len(methods) != initialMethodCount+1 {
				t.Fatalf("expected %d method(s), got %d", initialMethodCount+1, len(methods))
			}

			if diff := cmp.Diff(tt.expectedMethod, methods[len(methods)-1]); diff != "" {
				t.Errorf("method mismatch (-want +got):\n%s", diff)
			}

			if spec.StateMachineSpec == nil {
				t.Fatal("StateMachineSpec should not be nil after OnMessage call")
			}

			if len(spec.getRPCMethods()) == 0 {
				t.Error("RPC methods should be registered after OnMessage call")
			}
		})
	}
}

func TestAnalyzeMessage(t *testing.T) {
	tests := []struct {
		name     string
		instance AbstractMessage
		want     *message
	}{
		{
			name:     "analyzes TestRequest1 correctly",
			instance: &TestRequest1{},
			want: &message{
				Name: "TestRequest1",
				Fields: []field{
					{Name: "Data", Type: "string", Number: 1, IsRepeated: false},
				},
			},
		},
		{
			name:     "analyzes TestResponse1 correctly",
			instance: &TestResponse1{},
			want: &message{
				Name: "TestResponse1",
				Fields: []field{
					{Name: "Result", Type: "string", Number: 1, IsRepeated: false},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := analyzeMessage(tt.instance)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Errorf("analyzeMessage() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestMapGoField(t *testing.T) {
	tests := []struct {
		name          string
		goType        reflect.Type
		wantProtoType string
		wantRepeated  bool
	}{
		{
			name:          "maps string to string",
			goType:        reflect.TypeOf(""),
			wantProtoType: "string",
			wantRepeated:  false,
		},
		{
			name:          "maps bool to bool",
			goType:        reflect.TypeOf(false),
			wantProtoType: "bool",
			wantRepeated:  false,
		},
		{
			name:          "maps int64 to int64",
			goType:        reflect.TypeOf(int64(0)),
			wantProtoType: "int64",
			wantRepeated:  false,
		},
		{
			name:          "maps []string to repeated string",
			goType:        reflect.TypeOf([]string{}),
			wantProtoType: "string",
			wantRepeated:  true,
		},
		{
			name:          "handles unsupported type",
			goType:        reflect.TypeOf(make(chan int)),
			wantProtoType: "",
			wantRepeated:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotRepeated := mapGoField(tt.goType)
			if gotType != tt.wantProtoType {
				t.Errorf("mapGoField() type = %v, want %v", gotType, tt.wantProtoType)
			}
			if gotRepeated != tt.wantRepeated {
				t.Errorf("mapGoField() repeated = %v, want %v", gotRepeated, tt.wantRepeated)
			}
		})
	}
}
