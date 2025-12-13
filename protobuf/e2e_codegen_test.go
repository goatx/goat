package protobuf

import (
	"testing"

	"github.com/goatx/goat/internal/e2egen"
	"github.com/google/go-cmp/cmp"
)

func TestFormatStructLiteral(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		pkgAlias string
		typeName string
		data     map[string]any
		want     string
	}{
		{
			name:     "empty data",
			pkgAlias: "pb",
			typeName: "Request",
			data:     map[string]any{},
			want:     "&pb.Request{}",
		},
		{
			name:     "populated data",
			pkgAlias: "pb",
			typeName: "Response",
			data: map[string]any{
				"Count":  5,
				"Name":   "alice",
				"Active": true,
			},
			want: "&pb.Response{\n\t\t\t\tActive: true,\n\t\t\t\tCount: 5,\n\t\t\t\tName: \"alice\",\n\t\t\t}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatStructLiteral(tt.pkgAlias, tt.typeName, tt.data); got != tt.want {
				t.Fatalf("FormatStructLiteral() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatValue(t *testing.T) {
	t.Parallel()

	var nilPtr *int

	tests := []struct {
		name  string
		value any
		want  string
	}{
		{name: "string", value: "hello", want: "\"hello\""},
		{name: "bool", value: true, want: "true"},
		{name: "int", value: int64(42), want: "42"},
		{name: "uint", value: uint(7), want: "7"},
		{name: "float32", value: float32(3.5), want: "3.5"},
		{name: "float", value: 3.5, want: "3.5"},
		{name: "slice", value: []int{1, 2}, want: "[]int{1, 2}"},
		{name: "nil interface", value: nil, want: "nil"},
		{name: "nil pointer", value: nilPtr, want: "nil"},
		{name: "nil slice", value: []string(nil), want: "nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatValue(tt.value); got != tt.want {
				t.Fatalf("FormatValue() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGenerateMainTest(t *testing.T) {
	t.Parallel()

	suite := e2egen.TestSuite{
		TestPackage: "testpkg",
		Groups: []e2egen.TestGroup{
			{
				Name:          "User",
				SchemaPackage: "github.com/example/user",
				Operations:    nil,
			},
		},
	}

	ts := &testSuite{suite: suite}
	got, err := ts.generateMainTest()
	if err != nil {
		t.Fatalf("generateMainTest() error = %v", err)
	}

	want := `package testpkg

import (
	"log"
	"net"
	"os"
	"testing"

	pbuser "github.com/example/user"
	"google.golang.org/grpc"
)

var userClient pbuser.UserClient

func TestMain(m *testing.M) {
	lis0, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	grpcServer0 := grpc.NewServer()
	defer grpcServer0.Stop()
	// TODO: Register your service implementation here
	// pbuser.RegisterUserServer(grpcServer0, &yourServiceImplementation{})

	go func() {
		if err := grpcServer0.Serve(lis0); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	conn0, err := grpc.Dial(lis0.Addr().String(), grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn0.Close()

	userClient = pbuser.NewUserClient(conn0)

	os.Exit(m.Run())
}
`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("generateMainTest() mismatch (-want +got):\n%s", diff)
	}
}

func TestGenerateServiceTest(t *testing.T) {
	t.Parallel()

	suite := e2egen.TestSuite{
		TestPackage: "testpkg",
	}
	ts := &testSuite{suite: suite}

	group := e2egen.TestGroup{
		Name:          "User",
		SchemaPackage: "github.com/example/user",
		Operations: []e2egen.Operation{
			{
				Name: "CreateUser",
				TestCases: []e2egen.TestCase{
					{
						Name:       "case_0",
						InputType:  "CreateRequest",
						Input:      map[string]any{"Foo": "bar"},
						OutputType: "CreateResponse",
						Output:     map[string]any{"Ok": true},
					},
				},
			},
		},
	}

	got, err := ts.generateServiceTest(group)
	if err != nil {
		t.Fatalf("generateServiceTest() error = %v", err)
	}

	want := `package testpkg

import (
	"testing"

	pbuser "github.com/example/user"
	"github.com/google/go-cmp/cmp"
)

func TestCreateUser(t *testing.T) {
	tests := []struct {
		name     string
		input    *pbuser.CreateRequest
		expected *pbuser.CreateResponse
	}{
		{
			name: "case_0",
			input: &pbuser.CreateRequest{
				Foo: "bar",
			},
			expected: &pbuser.CreateResponse{
				Ok: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := userClient.CreateUser(t.Context(), tt.input)
			if err != nil {
				t.Fatalf("RPC call failed: %v", err)
			}
			if diff := cmp.Diff(tt.expected, actual); diff != "" {
				t.Errorf("CreateUser mismatch (-expected +actual):\n%s", diff)
			}
		})
	}
}
`

	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("generateServiceTest() mismatch (-want +got):\n%s", diff)
	}
}
