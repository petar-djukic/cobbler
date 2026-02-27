package inspect

import (
	"fmt"
	"testing"
)

func TestTranslationValidatorName(t *testing.T) {
	tv := NewTranslationValidator()
	if tv.Name() != "translation_validation" {
		t.Errorf("expected translation_validation, got %s", tv.Name())
	}
}

func TestTranslationValidatorFaultClass(t *testing.T) {
	tv := NewTranslationValidator()
	if tv.FaultClass() != "specification conformance errors" {
		t.Errorf("unexpected fault class: %s", tv.FaultClass())
	}
}

func TestTranslationValidatorNotApplicableWithoutCriteria(t *testing.T) {
	tv := NewTranslationValidator()
	input := &InspectInput{CrumbID: "test-1"}

	if tv.Applicable(input) {
		t.Error("expected not applicable with no criteria")
	}

	result, err := tv.Run(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictSkip {
		t.Errorf("expected skip, got %s", result.Verdict)
	}
}

func TestTranslationValidatorFileExistencePass(t *testing.T) {
	tv := &TranslationValidator{
		fileExists: func(path string) bool { return true },
		buildCheck: func(_ []string) error { return nil },
		testCheck:  func(_ []string) error { return nil },
	}

	input := &InspectInput{
		CrumbID:       "test-1",
		ModifiedFiles: []string{"main.go", "util.go"},
		PRDCriteria:   []string{"Files exist"},
	}

	result, err := tv.Run(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictPass {
		t.Errorf("expected pass, got %s", result.Verdict)
	}
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0, got %.3f", result.Score)
	}
}

func TestTranslationValidatorFileExistenceFail(t *testing.T) {
	tv := &TranslationValidator{
		fileExists: func(path string) bool { return path != "missing.go" },
		buildCheck: func(_ []string) error { return nil },
		testCheck:  func(_ []string) error { return nil },
	}

	input := &InspectInput{
		CrumbID:       "test-1",
		ModifiedFiles: []string{"main.go", "missing.go"},
		PRDCriteria:   []string{"Files exist"},
	}

	result, err := tv.Run(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictFail {
		t.Errorf("expected fail, got %s", result.Verdict)
	}
	// 1 pass (main.go) out of 2 file checks = 0.5.
	if result.Score != 0.5 {
		t.Errorf("expected score 0.5, got %.3f", result.Score)
	}
}

func TestTranslationValidatorCompilationAndTests(t *testing.T) {
	tv := &TranslationValidator{
		fileExists: func(_ string) bool { return true },
		buildCheck: func(_ []string) error { return nil },
		testCheck:  func(_ []string) error { return fmt.Errorf("test failure") },
	}

	input := &InspectInput{
		CrumbID:          "test-1",
		ModifiedFiles:    []string{"main.go"},
		ModifiedPackages: []string{"./cmd/cobbler"},
		PRDCriteria:      []string{"Code compiles and tests pass"},
	}

	result, err := tv.Run(input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Verdict != VerdictFail {
		t.Errorf("expected fail, got %s", result.Verdict)
	}
	// 1 file exists + 1 build passes + 0 test fail = 2/3.
	expected := 2.0 / 3.0
	if result.Score < expected-0.01 || result.Score > expected+0.01 {
		t.Errorf("expected score ~%.3f, got %.3f", expected, result.Score)
	}
}

func TestTranslationValidatorIsDeterministic(t *testing.T) {
	tv := NewTranslationValidator()
	input := &InspectInput{
		CrumbID:       "test-1",
		ModifiedFiles: []string{"main.go"},
		PRDCriteria:   []string{"File exists"},
	}

	// Override to avoid OS dependency.
	tv.fileExists = func(_ string) bool { return true }

	result, err := tv.Run(input)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Deterministic {
		t.Error("mechanical-only translation validation should be deterministic")
	}
}

func TestTranslationValidatorEvidenceRecorded(t *testing.T) {
	tv := &TranslationValidator{
		fileExists: func(path string) bool { return path == "exists.go" },
		buildCheck: func(_ []string) error { return nil },
		testCheck:  func(_ []string) error { return nil },
	}

	input := &InspectInput{
		CrumbID:       "test-1",
		ModifiedFiles: []string{"exists.go", "missing.go"},
		PRDCriteria:   []string{"Files present"},
	}

	result, err := tv.Run(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Evidence) != 2 {
		t.Fatalf("expected 2 evidence items, got %d", len(result.Evidence))
	}

	found := map[string]bool{"passed": false, "failed": false}
	for _, e := range result.Evidence {
		if e.CriterionID != "file_exists" {
			t.Errorf("unexpected criterion ID: %s", e.CriterionID)
		}
		if len(e.Detail) > 8 && e.Detail[:6] == "passed" {
			found["passed"] = true
		}
		if len(e.Detail) > 8 && e.Detail[:6] == "failed" {
			found["failed"] = true
		}
	}
	if !found["passed"] || !found["failed"] {
		t.Error("expected both passed and failed evidence entries")
	}
}
