package typeutil

import (
	"reflect"
	"strings"
)

func IsEventType(t reflect.Type) bool {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Kind() == reflect.Struct &&
		strings.HasPrefix(t.Name(), "Event[") &&
		t.PkgPath() == "github.com/goatx/goat"
}

func Name(v any) string {
	t := reflect.TypeOf(v)
	if t == nil {
		return ""
	}
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}

func NameOf[T any]() string {
	t := reflect.TypeFor[T]()
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.Name()
}
