package strcase

import "testing"

func TestToCamelCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"PascalCase", "PascalCase", "pascalCase"},
		{"UserID", "UserID", "userID"},
		{"SingleLower", "word", "word"},
		{"Empty", "", ""},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ToCamelCase(tt.input); got != tt.want {
				t.Fatalf("ToCamelCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToSnakeCase(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"Simple", "CamelCase", "camel_case"},
		{"SingleWord", "Camel", "camel"},
		{"Leading", "URLValue", "url_value"},
		{"TrailingUpper", "UserID", "user_id"},
		{"AcronymOnly", "URL", "url"},
		{"Empty", "", ""},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ToSnakeCase(tt.input); got != tt.want {
				t.Fatalf("ToSnakeCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToProtobufFieldName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"UserID", "UserID", "UserId"},
		{"HTTPServer", "HTTPServer", "HttpServer"},
		{"Simple", "Username", "Username"},
		{"URLValue", "URLValue", "UrlValue"},
		{"Empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ToProtobufFieldName(tt.input); got != tt.want {
				t.Fatalf("ToProtobufFieldName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
