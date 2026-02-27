package inspect

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func TestMutationRunnerName(t *testing.T) {
	mr := NewMutationRunner()
	if mr.Name() != "mutation_testing" {
		t.Errorf("expected mutation_testing, got %s", mr.Name())
	}
}

func TestMutationRunnerFaultClass(t *testing.T) {
	mr := NewMutationRunner()
	if mr.FaultClass() != "test suite inadequacy" {
		t.Errorf("unexpected fault class: %s", mr.FaultClass())
	}
}

func TestMutationRunnerNotApplicableForDocs(t *testing.T) {
	mr := NewMutationRunner()
	input := &InspectInput{
		WorkType:         "docs",
		ModifiedPackages: []string{"./pkg/foo"},
	}

	if mr.Applicable(input) {
		t.Error("expected not applicable for docs work type")
	}
}

func TestMutationRunnerNotApplicableWithoutPackages(t *testing.T) {
	mr := NewMutationRunner()
	input := &InspectInput{WorkType: "code"}

	if mr.Applicable(input) {
		t.Error("expected not applicable without modified packages")
	}
}

func TestMutationRunnerApplicableForCode(t *testing.T) {
	mr := NewMutationRunner()
	input := &InspectInput{
		WorkType:         "code",
		ModifiedPackages: []string{"./internal/inspect"},
	}

	if !mr.Applicable(input) {
		t.Error("expected applicable for code with packages")
	}
}

func TestMutationRunnerSkipsTestFiles(t *testing.T) {
	mr := &MutationRunner{
		runTests: func(_ []string) error { return nil },
	}

	input := &InspectInput{
		WorkType:         "code",
		ModifiedFiles:    []string{"foo_test.go"},
		ModifiedPackages: []string{"./pkg/foo"},
	}

	result, err := mr.Run(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictSkip {
		t.Errorf("expected skip for test-only files, got %s", result.Verdict)
	}
}

func TestFindMutationSites(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "example.go")
	code := `package example

func add(a, b int) int {
	if a < b {
		return a + b
	}
	return a - b
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	mr := NewMutationRunner()
	mutants, err := mr.findMutationSites(src)
	if err != nil {
		t.Fatal(err)
	}

	if len(mutants) == 0 {
		t.Fatal("expected mutation sites in example code")
	}

	// Should find: < (operator + boundary), + (operator), - (operator).
	types := make(map[MutationType]int)
	for _, m := range mutants {
		types[m.Type]++
	}
	if types[MutationOperatorReplace] == 0 {
		t.Error("expected operator replacement mutations")
	}
	if types[MutationBoundaryChange] == 0 {
		t.Error("expected boundary change mutations")
	}
}

func TestFindMutationSitesNegation(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "negate.go")
	code := `package example

func isNotEmpty(s string) bool {
	return !isEmpty(s)
}

func isEmpty(s string) bool {
	return len(s) == 0
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	mr := NewMutationRunner()
	mutants, err := mr.findMutationSites(src)
	if err != nil {
		t.Fatal(err)
	}

	found := false
	for _, m := range mutants {
		if m.Type == MutationConditionNegate {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected condition negation mutation for !isEmpty")
	}
}

func TestOperatorReplacement(t *testing.T) {
	tests := []struct {
		op       token.Token
		expected token.Token
		ok       bool
	}{
		{token.ADD, token.SUB, true},
		{token.SUB, token.ADD, true},
		{token.MUL, token.QUO, true},
		{token.EQL, token.NEQ, true},
		{token.LSS, token.GEQ, true},
		{token.LAND, token.LOR, true},
		{token.ASSIGN, token.ASSIGN, false}, // Not a replaceable operator.
	}

	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			got, ok := operatorReplacement(tt.op)
			if ok != tt.ok {
				t.Errorf("expected ok=%v, got %v", tt.ok, ok)
			}
			if ok && got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestBoundaryChange(t *testing.T) {
	tests := []struct {
		op       token.Token
		expected token.Token
		ok       bool
	}{
		{token.LSS, token.LEQ, true},
		{token.LEQ, token.LSS, true},
		{token.GTR, token.GEQ, true},
		{token.GEQ, token.GTR, true},
		{token.ADD, token.ADD, false},
	}

	for _, tt := range tests {
		t.Run(tt.op.String(), func(t *testing.T) {
			got, ok := boundaryChange(tt.op)
			if ok != tt.ok {
				t.Errorf("expected ok=%v, got %v", tt.ok, ok)
			}
			if ok && got != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, got)
			}
		})
	}
}

func TestMutationRunnerIsDeterministic(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "det.go")
	code := `package det

func add(a, b int) int { return a + b }
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}

	mr := &MutationRunner{
		runTests: func(_ []string) error { return nil },
	}

	input := &InspectInput{
		WorkType:         "code",
		ModifiedFiles:    []string{src},
		ModifiedPackages: []string{"./..."},
	}

	result, err := mr.Run(input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Deterministic {
		t.Error("mutation testing should be deterministic")
	}
}
