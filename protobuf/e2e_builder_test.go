package protobuf

import (
	"context"
	"testing"

	"github.com/goatx/goat"
	"github.com/goatx/goat/internal/e2egen"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestSerializeMessage(t *testing.T) {
	type sampleMessage struct {
		Message[*TestService1, *TestService1]

		Foo    string
		UserID string
		Count  int
		Event  goat.Event[*TestService1, *TestService1]
		hidden string
		_      int
	}

	msg := &sampleMessage{
		Foo:    "foo",
		UserID: "user-1",
		Count:  3,
		Event:  goat.Event[*TestService1, *TestService1]{},
		hidden: "secret",
	}

	t.Run("filters non-protobuf fields and converts field names", func(t *testing.T) {
		got := serializeMessage(msg)
		want := map[string]any{
			"Foo":    "foo",
			"UserId": "user-1",
			"Count":  3,
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("serializeMessage() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestBuildTestSuite(t *testing.T) {
	t.Run("builds suite with single service and method", func(t *testing.T) {
		opts := E2ETestOptions{
			ServiceSchemaPackage: "mypkg",
			Services: []ServiceTestCase{
				{
					Spec:        newTestSpec(),
					TestPackage: "example/pkg",
					Methods: []MethodTestCase{
						{
							MethodName: "Do",
							TestInputs: []AbstractMessage{
								&TestRequest1{Data: "foo"},
							},
						},
					},
				},
			},
		}

		got, err := buildTestSuite(opts)
		if err != nil {
			t.Fatalf("buildTestSuite() error = %v", err)
		}

		want := e2egen.TestSuite{
			TestPackage: "mypkg",
			Groups: []e2egen.TestGroup{
				{
					Name:          "TestService1",
					SchemaPackage: "example/pkg",
					Operations: []e2egen.Operation{
						{
							Name: "Do",
							TestCases: []e2egen.TestCase{
								{
									Name:       "case_0",
									InputType:  "TestRequest1",
									Input:      map[string]any{"Data": "foo"},
									OutputType: "TestResponse1",
									Output:     map[string]any{"Result": "foo_out"},
								},
							},
						},
					},
				},
			},
		}

		if diff := cmp.Diff(want, got); diff != "" {
			t.Fatalf("buildTestSuite() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestExecuteHandler(t *testing.T) {
	tests := []struct {
		name    string
		spec    AbstractServiceSpec
		method  string
		input   AbstractMessage
		want    AbstractMessage
		wantErr bool
	}{
		{
			name:   "success",
			spec:   newTestSpec(),
			method: "Do",
			input:  &TestRequest1{Data: "in"},
			want:   &TestResponse1{Result: "in_out"},
		},
		{
			name:    "missing handler",
			spec:    newTestSpec(),
			method:  "Nope",
			input:   &TestRequest1{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := executeHandler(tt.spec, tt.method, tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("executeHandler() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("executeHandler() error = %v", err)
			}

			if diff := cmp.Diff(tt.want, got,
				cmpopts.IgnoreUnexported(
					goat.Event[*TestService1, *TestService1]{},
				),
			); diff != "" {
				t.Fatalf("executeHandler() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func newTestSpec() *ServiceSpec[*TestService1] {
	spec := NewServiceSpec(&TestService1{})
	idle := &TestIdleState{}
	spec.DefineStates(idle).SetInitialState(idle)

	OnMessage(spec, idle, "Do",
		func(ctx context.Context, req *TestRequest1, sm *TestService1) Response[*TestResponse1] {
			return SendTo(ctx, sm, &TestResponse1{Result: req.Data + "_out"})
		})

	return spec
}
