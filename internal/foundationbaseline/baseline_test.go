package foundationbaseline

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var mandatoryFoundationPackages = []string{
	"internal/apierror",
	"internal/apicommand",
	"internal/apiquery",
	"internal/apiratelimit",
	"internal/authorization",
	"internal/authzpipeline",
	"internal/buildinfo",
	"internal/canonical",
	"internal/commandendpoint",
	"internal/config",
	"internal/foundationtest",
	"internal/health",
	"internal/identity",
	"internal/inbox",
	"internal/integrity",
	"internal/lease",
	"internal/orchestration",
	"internal/persistence/postgres",
	"internal/policy",
	"internal/repository",
	"internal/specification",
	"internal/toolregistry",
}

func TestMandatoryFoundationPackagesHaveUnitTests(t *testing.T) {
	t.Parallel()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) did not return the baseline test path")
	}
	packageDir := filepath.Dir(thisFile)
	repoRoot := filepath.Dir(filepath.Dir(packageDir))

	for _, relativeDir := range mandatoryFoundationPackages {
		relativeDir := relativeDir
		t.Run(relativeDir, func(t *testing.T) {
			t.Parallel()

			testFiles, err := filepath.Glob(filepath.Join(repoRoot, filepath.FromSlash(relativeDir), "*_test.go"))
			if err != nil {
				t.Fatalf("glob unit tests: %v", err)
			}
			if len(testFiles) == 0 {
				t.Fatalf("%s has no package-local *_test.go files", relativeDir)
			}

			hasTestFunction := false
			for _, testFile := range testFiles {
				parsed, err := parser.ParseFile(token.NewFileSet(), testFile, nil, parser.SkipObjectResolution)
				if err != nil {
					t.Fatalf("parse %s: %v", testFile, err)
				}
				for _, decl := range parsed.Decls {
					fn, ok := decl.(*ast.FuncDecl)
					if !ok || fn.Recv != nil {
						continue
					}
					if strings.HasPrefix(fn.Name.Name, "Test") && len(fn.Name.Name) > len("Test") {
						hasTestFunction = true
						break
					}
				}
				if hasTestFunction {
					break
				}
			}
			if !hasTestFunction {
				t.Fatalf("%s has *_test.go files but no top-level Test* function", relativeDir)
			}
		})
	}
}
