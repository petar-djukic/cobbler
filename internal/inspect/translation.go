// Translation validation technique for the inspect verification portfolio.
// Implements: prd008-inspect-verification R2 (Translation Validation).
package inspect

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// MechanicalCheck defines a single mechanical validation against an acceptance criterion.
type MechanicalCheck struct {
	CriterionID string                       // The AC or SC ID this check validates.
	Description string                       // Human-readable description.
	Check       func(input *InspectInput) bool // Returns true if the check passes.
}

// TranslationValidator checks stitch output against PRD acceptance criteria
// and use case success criteria using mechanical checks.
// Implements: prd008-inspect-verification R2.1-R2.4.
type TranslationValidator struct {
	fileExists  func(path string) bool
	buildCheck  func(packages []string) error
	testCheck   func(packages []string) error
}

// NewTranslationValidator creates a TranslationValidator with standard OS checks.
func NewTranslationValidator() *TranslationValidator {
	return &TranslationValidator{
		fileExists: fileExistsOS,
		buildCheck: buildPackages,
		testCheck:  testPackages,
	}
}

func (t *TranslationValidator) Name() string { return "translation_validation" }

func (t *TranslationValidator) FaultClass() string {
	return "specification conformance errors"
}

func (t *TranslationValidator) Applicable(input *InspectInput) bool {
	return len(input.PRDCriteria) > 0 || len(input.UCCriteria) > 0
}

// Run evaluates each acceptance criterion and success criterion mechanically.
// Semantic LLM-as-judge criteria are deferred; this implementation covers
// the deterministic mechanical subset (R2.2).
func (t *TranslationValidator) Run(input *InspectInput) (*TechniqueResult, error) {
	if !t.Applicable(input) {
		return &TechniqueResult{
			Name:          t.Name(),
			Score:         0,
			Verdict:       VerdictSkip,
			Deterministic: true,
		}, nil
	}

	checks := t.buildChecks(input)
	if len(checks) == 0 {
		return &TechniqueResult{
			Name:          t.Name(),
			Score:         0,
			Verdict:       VerdictSkip,
			Deterministic: true,
		}, nil
	}

	var passed int
	var evidence []Evidence

	for _, mc := range checks {
		ok := mc.Check(input)
		if ok {
			passed++
			evidence = append(evidence, Evidence{
				CriterionID: mc.CriterionID,
				Detail:      fmt.Sprintf("passed: %s", mc.Description),
			})
		} else {
			evidence = append(evidence, Evidence{
				CriterionID: mc.CriterionID,
				Detail:      fmt.Sprintf("failed: %s", mc.Description),
			})
		}
	}

	score := float64(passed) / float64(len(checks))
	verdict := VerdictPass
	if passed < len(checks) {
		verdict = VerdictFail
	}

	return &TechniqueResult{
		Name:          t.Name(),
		Score:         score,
		Verdict:       verdict,
		Evidence:      evidence,
		Deterministic: true, // Mechanical checks only; LLM judge is separate.
	}, nil
}

// buildChecks constructs mechanical checks from the available criteria and input.
func (t *TranslationValidator) buildChecks(input *InspectInput) []MechanicalCheck {
	var checks []MechanicalCheck

	// File existence checks: every modified file must exist.
	for _, f := range input.ModifiedFiles {
		checks = append(checks, MechanicalCheck{
			CriterionID: "file_exists",
			Description: fmt.Sprintf("file %s exists", f),
			Check: func(_ *InspectInput) bool {
				return t.fileExists(f)
			},
		})
	}

	// Build check: modified packages must compile.
	if len(input.ModifiedPackages) > 0 {
		checks = append(checks, MechanicalCheck{
			CriterionID: "compilation",
			Description: "modified packages compile",
			Check: func(in *InspectInput) bool {
				return t.buildCheck(in.ModifiedPackages) == nil
			},
		})
	}

	// Test check: tests in modified packages must pass.
	if len(input.ModifiedPackages) > 0 {
		checks = append(checks, MechanicalCheck{
			CriterionID: "tests_pass",
			Description: "tests pass in modified packages",
			Check: func(in *InspectInput) bool {
				return t.testCheck(in.ModifiedPackages) == nil
			},
		})
	}

	return checks
}

func fileExistsOS(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func buildPackages(packages []string) error {
	args := append([]string{"build"}, packages...)
	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func testPackages(packages []string) error {
	args := append([]string{"test"}, packages...)
	cmd := exec.Command("go", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tests failed: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}
