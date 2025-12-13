package typeutil

import (
	"reflect"
	"testing"

	"github.com/goatx/goat"
)

type sampleStruct struct{}

func TestName(t *testing.T) {
	t.Parallel()

	var value sampleStruct
	var nilPointer *sampleStruct

	tests := []struct {
		name  string
		input any
		want  string
	}{
		{name: "Nil", input: nil, want: ""},
		{name: "StructValue", input: value, want: "sampleStruct"},
		{name: "PointerValue", input: &value, want: "sampleStruct"},
		{name: "NilPointer", input: nilPointer, want: "sampleStruct"},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Name(tt.input); got != tt.want {
				t.Fatalf("Name(%v) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNameOf(t *testing.T) {
	t.Parallel()

	if got := NameOf[sampleStruct](); got != "sampleStruct" {
		t.Fatalf("NameOf[sampleStruct]() = %q, want %q", got, "sampleStruct")
	}
	if got := NameOf[*sampleStruct](); got != "sampleStruct" {
		t.Fatalf("NameOf[*sampleStruct]() = %q, want %q", got, "sampleStruct")
	}
}

type testStateMachine struct {
	goat.StateMachine
}
type otherEvent struct{}

func TestIsEventType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		typ  reflect.Type
		want bool
	}{
		{
			name: "StructValueFromGoatPackage",
			typ:  reflect.TypeOf(goat.Event[*testStateMachine, *testStateMachine]{}),
			want: true,
		},
		{
			name: "PointerValueFromGoatPackage",
			typ:  reflect.TypeOf(&goat.Event[*testStateMachine, *testStateMachine]{}),
			want: true,
		},
		{
			name: "PrefixMatchesButDifferentPackage",
			typ:  reflect.TypeOf(otherEvent{}),
			want: false,
		},
		{
			name: "StructInGoatPackageWithoutEventPrefix",
			typ:  reflect.TypeOf(goat.State{}),
			want: false,
		},
		{
			name: "NonStructType",
			typ:  reflect.TypeOf(42),
			want: false,
		},
		{
			name: "PointerToNonStruct",
			typ:  reflect.TypeOf(new(int)),
			want: false,
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsEventType(tt.typ); got != tt.want {
				t.Fatalf("IsEventType(%v) = %v, want %v", tt.typ, got, tt.want)
			}
		})
	}
}
