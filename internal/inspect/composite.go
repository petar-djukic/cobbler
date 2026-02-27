// Composite adequacy scorer for the inspect verification portfolio.
// Implements: prd008-inspect-verification R7 (Composite Adequacy Scoring).
package inspect

import "maps"

// Default scoring weights from prd008-inspect-verification R7.2.
var DefaultWeights = map[string]float64{
	"translation_validation": 0.30,
	"mutation_testing":       0.25,
	"differential_testing":   0.20,
	"property_based_testing": 0.15,
	"contract_injection":     0.10,
}

// Default action thresholds from prd008-inspect-verification R7.3.
const (
	DefaultAcceptThreshold = 0.80
	DefaultMendThreshold   = 0.50
)

// ScorerConfig holds configurable parameters for composite scoring.
type ScorerConfig struct {
	Weights          map[string]float64 // Technique name to weight.
	AcceptThreshold  float64            // Score >= this triggers accept.
	MendThreshold    float64            // Score >= this but < AcceptThreshold triggers mend.
	MinDeterministic float64            // Minimum fraction of weight from deterministic techniques.
}

// DefaultScorerConfig returns the default scorer configuration.
func DefaultScorerConfig() ScorerConfig {
	weights := make(map[string]float64, len(DefaultWeights))
	maps.Copy(weights, DefaultWeights)
	return ScorerConfig{
		Weights:          weights,
		AcceptThreshold:  DefaultAcceptThreshold,
		MendThreshold:    DefaultMendThreshold,
		MinDeterministic: 0.50,
	}
}

// Scorer computes composite adequacy scores from individual technique results.
type Scorer struct {
	config ScorerConfig
}

// NewScorer creates a scorer with the given configuration.
func NewScorer(config ScorerConfig) *Scorer {
	return &Scorer{config: config}
}

// Score computes the composite result from a set of technique results.
// Techniques with VerdictSkip are excluded from the weighted average.
// Returns a CompositeResult with ValidScore=false if fewer than two
// techniques produced non-skip results.
func (s *Scorer) Score(results []TechniqueResult) CompositeResult {
	cr := CompositeResult{
		TechniqueResults: results,
	}

	var weightedSum float64
	var totalWeight float64
	var scored int

	for _, r := range results {
		if r.Verdict == VerdictSkip {
			continue
		}
		scored++
		w := s.weightFor(r.Name)
		weightedSum += r.Score * w
		totalWeight += w
	}

	if scored < 2 {
		cr.ValidScore = false
		cr.Action = ActionHumanReview
		return cr
	}

	cr.ValidScore = true
	cr.CompositeScore = weightedSum / totalWeight
	cr.Action = s.actionFor(cr.CompositeScore)
	return cr
}

// DeterministicWeight returns the fraction of total weight assigned to
// deterministic techniques in the given results. Used to verify the
// prd008-inspect-verification R7.4 constraint.
func (s *Scorer) DeterministicWeight(results []TechniqueResult) float64 {
	var deterministicWeight float64
	var totalWeight float64

	for _, r := range results {
		if r.Verdict == VerdictSkip {
			continue
		}
		w := s.weightFor(r.Name)
		totalWeight += w
		if r.Deterministic {
			deterministicWeight += w
		}
	}

	if totalWeight == 0 {
		return 0
	}
	return deterministicWeight / totalWeight
}

func (s *Scorer) weightFor(name string) float64 {
	if w, ok := s.config.Weights[name]; ok {
		return w
	}
	// Unknown techniques get equal share of remaining weight.
	return 0.10
}

func (s *Scorer) actionFor(score float64) Action {
	if score >= s.config.AcceptThreshold {
		return ActionAccept
	}
	if score >= s.config.MendThreshold {
		return ActionMend
	}
	return ActionHumanReview
}
