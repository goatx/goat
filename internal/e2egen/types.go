package e2egen

type TestSuite struct {
	TestPackage string
	Groups      []TestGroup
}

type TestGroup struct {
	Name          string
	SchemaPackage string
	Operations    []Operation
}

type Operation struct {
	Name      string
	TestCases []TestCase
}

type TestCase struct {
	Name       string
	InputType  string
	Input      map[string]any
	OutputType string
	Output     map[string]any
}
