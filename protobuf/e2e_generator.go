package protobuf

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/goatx/goat/internal/strcase"
)

const (
	defaultE2EOutputDir         = "./tests"
	defaultServiceSchemaPackage = "main"
)

type MethodTestCase struct {
	MethodName string
	TestInputs []AbstractMessage
}

type ServiceTestCase struct {
	Spec        AbstractServiceSpec
	TestPackage string
	Methods     []MethodTestCase
}

type E2ETestOptions struct {
	OutputDir            string
	ServiceSchemaPackage string
	Services             []ServiceTestCase
}

// GenerateE2ETestSuite generates a complete E2E test suite including both main_test.go and all service-specific test files.
//
// This function creates:
//   - main_test.go: Contains test suite setup, fixtures, and test execution framework
//   - <service_name>_test.go: Individual test files for each service with method-specific test cases
//
// Use this function when you need a complete test suite with both infrastructure and test cases.
// For partial generation, see GenerateE2EMainTest (infrastructure only) or GenerateE2EServiceTests (test cases only).
//
// The function applies default values:
//   - OutputDir defaults to "./tests" if not specified
//   - ServiceSchemaPackage defaults to "main" if not specified
//
// Example:
//
//	err := GenerateE2ETestSuite(E2ETestOptions{
//	    OutputDir:            "./e2e",
//	    ServiceSchemaPackage: "e2e_test",
//	    Services: []ServiceTestCase{
//	        {Spec: userServiceSpec, TestPackage: "user", Methods: userMethods},
//	    },
//	})
//
// Returns an error if directory creation, test generation, or file writing fails.
func GenerateE2ETestSuite(opts E2ETestOptions) error {
	if opts.OutputDir == "" {
		opts.OutputDir = defaultE2EOutputDir
	}
	if opts.ServiceSchemaPackage == "" {
		opts.ServiceSchemaPackage = defaultServiceSchemaPackage
	}

	if err := os.MkdirAll(opts.OutputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	suite, err := buildTestSuite(opts)
	if err != nil {
		return err
	}

	testSuite := &testSuite{suite: suite}

	mainTest, err := testSuite.generateMainTest()
	if err != nil {
		return fmt.Errorf("failed to generate main_test.go: %w", err)
	}

	mainPath := filepath.Join(opts.OutputDir, "main_test.go")
	if err := os.WriteFile(mainPath, []byte(mainTest), 0o600); err != nil {
		return fmt.Errorf("failed to write main_test.go: %w", err)
	}

	for _, group := range suite.Groups {
		serviceTest, err := testSuite.generateServiceTest(group)
		if err != nil {
			return fmt.Errorf("failed to generate test for %s: %w", group.Name, err)
		}

		filename := strcase.ToSnakeCase(group.Name) + "_test.go"
		outputPath := filepath.Join(opts.OutputDir, filename)
		if err := os.WriteFile(outputPath, []byte(serviceTest), 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}

// GenerateE2EServiceTests generates only the service-specific test files without main_test.go.
//
// This function creates:
//   - <service_name>_test.go: Individual test files for each service with method-specific test cases
//
// Use this function when you already have main_test.go (test infrastructure) and only need to add or update service test cases.
// For complete test suite generation, see GenerateE2ETestSuite. For infrastructure only, see GenerateE2EMainTest.
//
// The function applies default values:
//   - OutputDir defaults to "./tests" if not specified
//   - ServiceSchemaPackage defaults to "main" if not specified
//
// Example:
//
//	err := GenerateE2EServiceTests(E2ETestOptions{
//	    OutputDir:            "./e2e",
//	    ServiceSchemaPackage: "e2e_test",
//	    Services: []ServiceTestCase{
//	        {Spec: orderServiceSpec, TestPackage: "order", Methods: orderMethods},
//	    },
//	})
//
// Returns an error if directory creation, test generation, or file writing fails.
func GenerateE2EServiceTests(opts E2ETestOptions) error {
	if opts.OutputDir == "" {
		opts.OutputDir = defaultE2EOutputDir
	}
	if opts.ServiceSchemaPackage == "" {
		opts.ServiceSchemaPackage = defaultServiceSchemaPackage
	}

	if err := os.MkdirAll(opts.OutputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	suite, err := buildTestSuite(opts)
	if err != nil {
		return err
	}

	ts := &testSuite{suite: suite}

	for _, group := range suite.Groups {
		code, err := ts.generateServiceTest(group)
		if err != nil {
			return fmt.Errorf("failed to generate test for %s: %w", group.Name, err)
		}

		filename := strcase.ToSnakeCase(group.Name) + "_test.go"
		outputPath := filepath.Join(opts.OutputDir, filename)
		if err := os.WriteFile(outputPath, []byte(code), 0o600); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}

// GenerateE2EMainTest generates only the main_test.go file without service-specific test files.
//
// This function creates:
//   - main_test.go: Contains test suite setup, fixtures, and test execution framework
//
// Use this function when you need to set up or update the test infrastructure without generating service test cases.
// For complete test suite generation, see GenerateE2ETestSuite. For test cases only, see GenerateE2EServiceTests.
//
// The function applies default values:
//   - OutputDir defaults to "./tests" if not specified
//   - ServiceSchemaPackage defaults to "main" if not specified
//
// Example:
//
//	err := GenerateE2EMainTest(E2ETestOptions{
//	    OutputDir:            "./e2e",
//	    ServiceSchemaPackage: "e2e_test",
//	    Services: []ServiceTestCase{
//	        {Spec: userServiceSpec, TestPackage: "user", Methods: nil},
//	    },
//	})
//
// Returns an error if directory creation, test generation, or file writing fails.
func GenerateE2EMainTest(opts E2ETestOptions) error {
	if opts.OutputDir == "" {
		opts.OutputDir = defaultE2EOutputDir
	}
	if opts.ServiceSchemaPackage == "" {
		opts.ServiceSchemaPackage = defaultServiceSchemaPackage
	}
	if err := os.MkdirAll(opts.OutputDir, 0o750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	suite, err := buildTestSuite(opts)
	if err != nil {
		return err
	}

	ts := &testSuite{suite: suite}
	code, err := ts.generateMainTest()
	if err != nil {
		return fmt.Errorf("failed to generate main_test.go: %w", err)
	}

	mainPath := filepath.Join(opts.OutputDir, "main_test.go")
	if err := os.WriteFile(mainPath, []byte(code), 0o600); err != nil {
		return fmt.Errorf("failed to write main_test.go: %w", err)
	}

	return nil
}
