package protobuf

import (
	"fmt"
	"go/format"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/goatx/goat/internal/e2egen"
	"github.com/goatx/goat/internal/strcase"
)

type testSuite struct {
	suite e2egen.TestSuite
}

func (s *testSuite) generateMainTest() (string, error) {
	var b strings.Builder

	b.WriteString("package ")
	b.WriteString(s.suite.TestPackage)
	b.WriteString("\n\n")

	b.WriteString("import (\n\t\"log\"\n\t\"net\"\n\t\"os\"\n\t\"testing\"\n\n\t\"google.golang.org/grpc\"\n")
	seen := make(map[string]bool)
	for _, group := range s.suite.Groups {
		if seen[group.SchemaPackage] {
			continue
		}
		b.WriteString("\tpb")
		b.WriteString(strcase.ToSnakeCase(group.Name))
		b.WriteString(" \"")
		b.WriteString(group.SchemaPackage)
		b.WriteString("\"\n")
		seen[group.SchemaPackage] = true
	}
	b.WriteString(")\n\n")

	for _, group := range s.suite.Groups {
		snake := strcase.ToSnakeCase(group.Name)
		clientVar := snake + "Client"
		b.WriteString("var ")
		b.WriteString(clientVar)
		b.WriteString(" pb")
		b.WriteString(snake)
		b.WriteString(".")
		b.WriteString(group.Name)
		b.WriteString("Client\n")
	}

	b.WriteString("\nfunc TestMain(m *testing.M) {\n")
	for i, group := range s.suite.Groups {
		iStr := strconv.Itoa(i)
		snake := strcase.ToSnakeCase(group.Name)
		clientVar := snake + "Client"

		b.WriteString("\tlis")
		b.WriteString(iStr)
		b.WriteString(", err := net.Listen(\"tcp\", \"localhost:0\")\n")
		b.WriteString("\tif err != nil {\n\t\tlog.Fatalf(\"Failed to listen: %v\", err)\n\t}\n\n")

		b.WriteString("\tgrpcServer")
		b.WriteString(iStr)
		b.WriteString(" := grpc.NewServer()\n")
		b.WriteString("\tdefer grpcServer")
		b.WriteString(iStr)
		b.WriteString(".Stop()\n")
		b.WriteString("\t// TODO: Register your service implementation here\n")
		b.WriteString("\t// pb")
		b.WriteString(snake)
		b.WriteString(".Register")
		b.WriteString(group.Name)
		b.WriteString("Server(grpcServer")
		b.WriteString(iStr)
		b.WriteString(", &yourServiceImplementation{})\n\n")

		b.WriteString("\tgo func() {\n\t\tif err := grpcServer")
		b.WriteString(iStr)
		b.WriteString(".Serve(lis")
		b.WriteString(iStr)
		b.WriteString("); err != nil {\n\t\t\tlog.Fatalf(\"Failed to serve: %v\", err)\n\t\t}\n\t}()\n\n")

		b.WriteString("\tconn")
		b.WriteString(iStr)
		b.WriteString(", err := grpc.Dial(lis")
		b.WriteString(iStr)
		b.WriteString(".Addr().String(), grpc.WithInsecure())\n")
		b.WriteString("\tif err != nil {\n\t\tlog.Fatalf(\"Failed to dial: %v\", err)\n\t}\n")
		b.WriteString("\tdefer conn")
		b.WriteString(iStr)
		b.WriteString(".Close()\n\n")

		b.WriteString("\t")
		b.WriteString(clientVar)
		b.WriteString(" = pb")
		b.WriteString(snake)
		b.WriteString(".New")
		b.WriteString(group.Name)
		b.WriteString("Client(conn")
		b.WriteString(iStr)
		b.WriteString(")\n\n")
	}

	b.WriteString("\tos.Exit(m.Run())\n}\n")

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return b.String(), fmt.Errorf("failed to format: %w", err)
	}
	return string(formatted), nil
}

func (s *testSuite) generateServiceTest(group e2egen.TestGroup) (string, error) {
	var b strings.Builder
	snake := strcase.ToSnakeCase(group.Name)

	b.WriteString("package ")
	b.WriteString(s.suite.TestPackage)
	b.WriteString("\n\nimport (\n\t\"testing\"\n\n\t\"github.com/google/go-cmp/cmp\"\n")
	b.WriteString("\tpb")
	b.WriteString(snake)
	b.WriteString(" \"")
	b.WriteString(group.SchemaPackage)
	b.WriteString("\"\n")
	b.WriteString(")\n\n")

	for _, op := range group.Operations {
		if err := s.writeMethodTest(&b, group, op); err != nil {
			return "", err
		}
	}

	formatted, err := format.Source([]byte(b.String()))
	if err != nil {
		return b.String(), fmt.Errorf("failed to format: %w", err)
	}
	return string(formatted), nil
}

func (*testSuite) writeMethodTest(b *strings.Builder, group e2egen.TestGroup, op e2egen.Operation) error {
	if len(op.TestCases) == 0 {
		return fmt.Errorf("no test cases for method %s", op.Name)
	}

	first := op.TestCases[0]
	snake := strcase.ToSnakeCase(group.Name)
	alias := "pb" + snake
	clientVar := snake + "Client"

	b.WriteString("func Test")
	b.WriteString(op.Name)
	b.WriteString("(t *testing.T) {\n")

	b.WriteString("\ttests := []struct {\n\t\tname     string\n\t\tinput    *")
	b.WriteString(alias)
	b.WriteString(".")
	b.WriteString(first.InputType)
	b.WriteString("\n\t\texpected *")
	b.WriteString(alias)
	b.WriteString(".")
	b.WriteString(first.OutputType)
	b.WriteString("\n\t}{\n")

	for _, tc := range op.TestCases {
		b.WriteString("\t\t{\n")
		b.WriteString("\t\t\tname: \"")
		b.WriteString(tc.Name)
		b.WriteString("\",\n")
		b.WriteString("\t\t\tinput: ")
		b.WriteString(formatStructLiteral(alias, tc.InputType, tc.Input))
		b.WriteString(",\n")
		b.WriteString("\t\t\texpected: ")
		b.WriteString(formatStructLiteral(alias, tc.OutputType, tc.Output))
		b.WriteString(",\n")
		b.WriteString("\t\t},\n")
	}

	b.WriteString("\t}\n\n\tfor _, tt := range tests {\n\t\tt.Run(tt.name, func(t *testing.T) {\n")

	b.WriteString("\t\t\tactual, err := ")
	b.WriteString(clientVar)
	b.WriteString(".")
	b.WriteString(op.Name)
	b.WriteString("(t.Context(), tt.input)\n")

	b.WriteString("\t\t\tif err != nil {\n\t\t\t\tt.Fatalf(\"RPC call failed: %v\", err)\n\t\t\t}\n")

	b.WriteString("\t\t\tif diff := cmp.Diff(tt.expected, actual); diff != \"\" {\n\t\t\t\tt.Errorf(\"")
	b.WriteString(op.Name)
	b.WriteString(" mismatch (-expected +actual):\\n%s\", diff)\n\t\t\t}\n")

	b.WriteString("\t\t})\n\t}\n}\n\n")

	return nil
}

func formatStructLiteral(pkgAlias, typeName string, data map[string]any) string {
	if len(data) == 0 {
		return "&" + pkgAlias + "." + typeName + "{}"
	}

	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	b.WriteString("&")
	b.WriteString(pkgAlias)
	b.WriteString(".")
	b.WriteString(typeName)
	b.WriteString("{\n")

	for _, k := range keys {
		b.WriteString("\t\t\t\t")
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(formatValue(data[k]))
		b.WriteString(",\n")
	}
	b.WriteString("\t\t\t}")
	return b.String()
}

func formatValue(value any) string {
	if value == nil {
		return "nil"
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr, reflect.Slice, reflect.Map:
		if v.IsNil() {
			return "nil"
		}
	}

	switch v.Kind() {
	case reflect.String:
		return strconv.Quote(v.String())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(v.Float(), 'g', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	default:
		return fmt.Sprintf("%#v", value)
	}
}
