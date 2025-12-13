package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/goatx/goat/protobuf"
	"github.com/google/go-cmp/cmp"
)

func TestUserServiceProtobufGeneration(t *testing.T) {
	spec := createUserServiceModel()
	tempDir := t.TempDir()
	outputPath := filepath.Join(tempDir, "proto")

	opts := protobuf.GenerateOptions{
		OutputDir:   outputPath,
		PackageName: "user.service",
		GoPackage:   "github.com/goatx/goat/protobuf/example/proto",
		Filename:    "user_service.proto",
	}

	err := protobuf.Generate(opts, spec)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	protoFile := filepath.Join(outputPath, "user_service.proto")
	// #nosec G304 - protoFile is constructed from t.TempDir() and fixed name in test
	content, err := os.ReadFile(protoFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	got := string(content)

	goldenPath := filepath.Join("proto", "user_service.proto.golden")
	// #nosec G304 - fixed golden path within repo
	wantBytes, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("ReadFile() golden error = %v", err)
	}
	want := string(wantBytes)

	if got != want {
		t.Errorf("Generated proto content mismatch.\nGot:\n%s\nWant:\n%s", got, want)
	}
}

func TestUserServiceE2EGeneration(t *testing.T) {
	spec := createUserServiceModel()
	outputPath := filepath.Join(t.TempDir(), "e2e")

	opts := protobuf.E2ETestOptions{
		OutputDir:            outputPath,
		ServiceSchemaPackage: "main",
		Services: []protobuf.ServiceTestCase{
			{
				Spec:        spec,
				TestPackage: "github.com/goatx/goat/protobuf/example/proto",
				Methods: []protobuf.MethodTestCase{
					{
						MethodName: "CreateUser",
						TestInputs: []protobuf.AbstractMessage{
							&CreateUserRequest{Username: "alice", Email: "alice@example.com"},
							&CreateUserRequest{Username: "bob", Email: "bob@example.com"},
						},
					},
					{
						MethodName: "GetUser",
						TestInputs: []protobuf.AbstractMessage{
							&GetUserRequest{UserID: "user_123"},
						},
					},
				},
			},
		},
	}

	err := protobuf.GenerateE2ETestSuite(opts)
	if err != nil {
		t.Fatalf("GenerateE2ETestSuite() error = %v", err)
	}

	goldenFiles := []struct {
		generated string
		golden    string
	}{
		{"main_test.go", "main_test.go.golden"},
		{"user_service_test.go", "user_service_test.go.golden"},
	}

	for _, gf := range goldenFiles {
		generatedPath := filepath.Join(outputPath, gf.generated)
		// #nosec G304 - generatedPath is constructed from t.TempDir() and fixed name in test
		gotBytes, err := os.ReadFile(generatedPath)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", gf.generated, err)
		}

		goldenPath := filepath.Join("e2e", gf.golden)
		// #nosec G304 - fixed golden path within repo
		wantBytes, err := os.ReadFile(goldenPath)
		if err != nil {
			t.Fatalf("ReadFile(%s) golden error = %v", gf.golden, err)
		}

		if diff := cmp.Diff(string(wantBytes), string(gotBytes)); diff != "" {
			t.Errorf("%s mismatch (-want +got):\n%s", gf.generated, diff)
		}
	}
}
