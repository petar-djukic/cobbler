// Mutation testing technique for the inspect verification portfolio.
// Implements: prd008-inspect-verification R3 (Mutation Testing).
package inspect

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
)

// MutationType describes the kind of syntactic mutation applied.
type MutationType string

const (
	MutationOperatorReplace   MutationType = "operator_replacement"
	MutationConditionNegate   MutationType = "condition_negation"
	MutationBoundaryChange    MutationType = "boundary_change"
	MutationStatementDelete   MutationType = "statement_deletion"
)

// Mutant represents a single injected fault in source code.
type Mutant struct {
	FilePath     string       // Source file containing the mutation.
	Line         int          // Line number of the mutation.
	Type         MutationType // Kind of mutation applied.
	Original     string       // Original code fragment.
	Mutated      string       // Mutated code fragment.
	Killed       bool         // Whether tests detected this mutant.
	Equivalent   bool         // Whether the mutant is semantically equivalent.
	KillingTest  string       // Test that detected the mutant (if killed).
}

// MutationRunner injects syntactic mutations and checks test detection.
// Implements: prd008-inspect-verification R3.1-R3.3.
type MutationRunner struct {
	runTests func(packages []string) error
}

// NewMutationRunner creates a MutationRunner with standard test execution.
func NewMutationRunner() *MutationRunner {
	return &MutationRunner{
		runTests: testPackages,
	}
}

func (m *MutationRunner) Name() string { return "mutation_testing" }

func (m *MutationRunner) FaultClass() string {
	return "test suite inadequacy"
}

func (m *MutationRunner) Applicable(input *InspectInput) bool {
	return input.WorkType == "code" && len(input.ModifiedPackages) > 0
}

// Run executes mutation testing against modified packages.
// For each Go source file in the modified packages, it identifies mutation
// sites, applies mutations one at a time, runs tests, and records whether
// each mutant was killed.
func (m *MutationRunner) Run(input *InspectInput) (*TechniqueResult, error) {
	if !m.Applicable(input) {
		return &TechniqueResult{
			Name:          m.Name(),
			Score:         0,
			Verdict:       VerdictSkip,
			Deterministic: true,
		}, nil
	}

	var allMutants []Mutant
	for _, file := range input.ModifiedFiles {
		if !strings.HasSuffix(file, ".go") || strings.HasSuffix(file, "_test.go") {
			continue
		}
		mutants, err := m.findMutationSites(file)
		if err != nil {
			continue // Skip files we cannot parse.
		}
		allMutants = append(allMutants, mutants...)
	}

	if len(allMutants) == 0 {
		return &TechniqueResult{
			Name:          m.Name(),
			Score:         0,
			Verdict:       VerdictSkip,
			Deterministic: true,
		}, nil
	}

	// Apply each mutation, run tests, restore.
	for i := range allMutants {
		allMutants[i].Killed = m.applyAndTest(
			allMutants[i].FilePath,
			allMutants[i].Line,
			allMutants[i].Original,
			allMutants[i].Mutated,
			input.ModifiedPackages,
		)
	}

	var killed, total int
	var evidence []Evidence
	for _, mut := range allMutants {
		if mut.Equivalent {
			continue
		}
		total++
		if mut.Killed {
			killed++
		} else {
			evidence = append(evidence, Evidence{
				FilePath: mut.FilePath,
				Detail: fmt.Sprintf(
					"surviving mutant at line %d: %s → %s (%s)",
					mut.Line, mut.Original, mut.Mutated, mut.Type,
				),
			})
		}
	}

	if total == 0 {
		return &TechniqueResult{
			Name:          m.Name(),
			Score:         0,
			Verdict:       VerdictSkip,
			Deterministic: true,
		}, nil
	}

	score := float64(killed) / float64(total)
	verdict := VerdictPass
	if score < 1.0 {
		verdict = VerdictFail
	}

	return &TechniqueResult{
		Name:          m.Name(),
		Score:         score,
		Verdict:       verdict,
		Evidence:      evidence,
		Deterministic: true,
	}, nil
}

// findMutationSites parses a Go file and identifies mutation candidates.
// Returns a list of potential mutants without applying them.
func (m *MutationRunner) findMutationSites(filePath string) ([]Mutant, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", filePath, err)
	}

	var mutants []Mutant

	ast.Inspect(f, func(n ast.Node) bool {
		switch expr := n.(type) {
		case *ast.BinaryExpr:
			if replacement, ok := operatorReplacement(expr.Op); ok {
				mutants = append(mutants, Mutant{
					FilePath: filePath,
					Line:     fset.Position(expr.Pos()).Line,
					Type:     MutationOperatorReplace,
					Original: expr.Op.String(),
					Mutated:  replacement.String(),
				})
			}
			if boundary, ok := boundaryChange(expr.Op); ok {
				mutants = append(mutants, Mutant{
					FilePath: filePath,
					Line:     fset.Position(expr.Pos()).Line,
					Type:     MutationBoundaryChange,
					Original: expr.Op.String(),
					Mutated:  boundary.String(),
				})
			}
		case *ast.UnaryExpr:
			if expr.Op == token.NOT {
				mutants = append(mutants, Mutant{
					FilePath: filePath,
					Line:     fset.Position(expr.Pos()).Line,
					Type:     MutationConditionNegate,
					Original: "!expr",
					Mutated:  "expr",
				})
			}
		}
		return true
	})

	return mutants, nil
}

// applyAndTest applies a mutation, runs tests, and restores the original file.
// Returns true if the mutation was detected (killed).
func (m *MutationRunner) applyAndTest(filePath string, line int, original, mutated string, packages []string) bool {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false
	}

	lines := strings.Split(string(content), "\n")
	if line < 1 || line > len(lines) {
		return false
	}

	originalLine := lines[line-1]
	mutatedLine := strings.Replace(originalLine, original, mutated, 1)
	if mutatedLine == originalLine {
		return false // Mutation did not apply; skip.
	}

	lines[line-1] = mutatedLine
	if err := os.WriteFile(filePath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return false
	}

	// Run tests. If they fail, the mutant was killed.
	killed := m.runTests(packages) != nil

	// Restore original.
	_ = os.WriteFile(filePath, content, 0o644)

	return killed
}

// operatorReplacement returns a replacement operator for arithmetic and comparison operators.
func operatorReplacement(op token.Token) (token.Token, bool) {
	replacements := map[token.Token]token.Token{
		token.ADD: token.SUB,
		token.SUB: token.ADD,
		token.MUL: token.QUO,
		token.QUO: token.MUL,
		token.EQL: token.NEQ,
		token.NEQ: token.EQL,
		token.LSS: token.GEQ,
		token.GEQ: token.LSS,
		token.GTR: token.LEQ,
		token.LEQ: token.GTR,
		token.LAND: token.LOR,
		token.LOR:  token.LAND,
	}
	r, ok := replacements[op]
	return r, ok
}

// boundaryChange returns a boundary-shifted operator.
func boundaryChange(op token.Token) (token.Token, bool) {
	changes := map[token.Token]token.Token{
		token.LSS: token.LEQ,
		token.LEQ: token.LSS,
		token.GTR: token.GEQ,
		token.GEQ: token.GTR,
	}
	r, ok := changes[op]
	return r, ok
}

