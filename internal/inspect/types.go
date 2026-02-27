// Package inspect implements the verification portfolio for evaluating stitch output.
// Implements: prd008-inspect-verification R1 (Technique Interface).
package inspect

import "fmt"

// Verdict represents the outcome of a verification technique.
type Verdict string

const (
	VerdictPass Verdict = "pass"
	VerdictFail Verdict = "fail"
	VerdictSkip Verdict = "skip"
)

// Action represents the composite scorer's recommended action.
type Action string

const (
	ActionAccept      Action = "accept"
	ActionMend        Action = "mend"
	ActionHumanReview Action = "human_review"
)

// Evidence records a single piece of verification evidence.
type Evidence struct {
	CriterionID string // PRD criterion or use case success criterion ID.
	FilePath    string // File path relevant to the evidence.
	Detail      string // Description of what was found.
}

// TechniqueResult is the typed result returned by each verification technique.
// Implements: prd008-inspect-verification R1.2.
type TechniqueResult struct {
	Name          string     // Technique name (e.g., "translation_validation").
	Score         float64    // Numeric score from 0.0 to 1.0.
	Verdict       Verdict    // Pass, fail, or skip.
	Evidence      []Evidence // Supporting evidence for the verdict.
	Deterministic bool       // Whether the technique is fully deterministic.
}

// Technique is the interface that each verification technique implements.
// Implements: prd008-inspect-verification R1.1.
type Technique interface {
	// Name returns the technique identifier.
	Name() string

	// FaultClass returns a description of the fault class this technique targets.
	FaultClass() string

	// Applicable reports whether the technique can run given the available inputs.
	Applicable(input *InspectInput) bool

	// Run executes the technique and returns a typed result.
	Run(input *InspectInput) (*TechniqueResult, error)
}

// InspectInput gathers the inputs available to verification techniques.
type InspectInput struct {
	CrumbID          string            // ID of the crumb being inspected.
	WorkType         string            // Work type (code, docs).
	ModifiedFiles    []string          // Files modified by stitch.
	ModifiedPackages []string          // Go packages modified by stitch.
	Diff             string            // Unified diff of stitch output.
	PRDCriteria      []string          // Acceptance criteria from the driving PRD.
	UCCriteria       []string          // Success criteria from the driving use case.
	FixtureDir       string            // Directory containing benchmark fixtures.
	PRDRequirements  map[string]string // Requirement ID to requirement text.
}

// CompositeResult aggregates technique results into a final verdict.
// Implements: prd008-inspect-verification R7.
type CompositeResult struct {
	TechniqueResults []TechniqueResult // Individual technique results.
	CompositeScore   float64           // Weighted average of available scores.
	Action           Action            // Recommended action based on thresholds.
	ValidScore       bool              // False if fewer than two techniques produced results.
}

// Error wrapping for inspect context.
var (
	ErrInsufficientTechniques = fmt.Errorf("inspect: fewer than two techniques produced results")
	ErrInvalidWeight          = fmt.Errorf("inspect: technique weight must be between 0.0 and 1.0")
	ErrNoTechniques           = fmt.Errorf("inspect: no techniques registered")
)
