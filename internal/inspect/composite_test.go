package inspect

import (
	"math"
	"testing"
)

func TestScorerAcceptAction(t *testing.T) {
	scorer := NewScorer(DefaultScorerConfig())
	results := []TechniqueResult{
		{Name: "translation_validation", Score: 0.90, Verdict: VerdictPass, Deterministic: false},
		{Name: "mutation_testing", Score: 0.85, Verdict: VerdictPass, Deterministic: true},
		{Name: "differential_testing", Score: 0.80, Verdict: VerdictPass, Deterministic: true},
	}

	cr := scorer.Score(results)

	if !cr.ValidScore {
		t.Fatal("expected valid score with 3 non-skip results")
	}
	if cr.Action != ActionAccept {
		t.Errorf("expected accept, got %s (score=%.3f)", cr.Action, cr.CompositeScore)
	}
	if cr.CompositeScore < DefaultAcceptThreshold {
		t.Errorf("expected score >= %.2f, got %.3f", DefaultAcceptThreshold, cr.CompositeScore)
	}
}

func TestScorerMendAction(t *testing.T) {
	scorer := NewScorer(DefaultScorerConfig())
	results := []TechniqueResult{
		{Name: "translation_validation", Score: 0.60, Verdict: VerdictFail, Deterministic: false},
		{Name: "mutation_testing", Score: 0.70, Verdict: VerdictFail, Deterministic: true},
	}

	cr := scorer.Score(results)

	if !cr.ValidScore {
		t.Fatal("expected valid score with 2 non-skip results")
	}
	if cr.Action != ActionMend {
		t.Errorf("expected mend, got %s (score=%.3f)", cr.Action, cr.CompositeScore)
	}
}

func TestScorerHumanReviewAction(t *testing.T) {
	scorer := NewScorer(DefaultScorerConfig())
	results := []TechniqueResult{
		{Name: "translation_validation", Score: 0.20, Verdict: VerdictFail, Deterministic: false},
		{Name: "mutation_testing", Score: 0.30, Verdict: VerdictFail, Deterministic: true},
	}

	cr := scorer.Score(results)

	if !cr.ValidScore {
		t.Fatal("expected valid score with 2 non-skip results")
	}
	if cr.Action != ActionHumanReview {
		t.Errorf("expected human_review, got %s (score=%.3f)", cr.Action, cr.CompositeScore)
	}
	if cr.CompositeScore >= DefaultMendThreshold {
		t.Errorf("expected score < %.2f, got %.3f", DefaultMendThreshold, cr.CompositeScore)
	}
}

func TestScorerSkipExcluded(t *testing.T) {
	scorer := NewScorer(DefaultScorerConfig())
	results := []TechniqueResult{
		{Name: "translation_validation", Score: 0.90, Verdict: VerdictPass, Deterministic: false},
		{Name: "mutation_testing", Score: 0.85, Verdict: VerdictPass, Deterministic: true},
		{Name: "differential_testing", Score: 0.0, Verdict: VerdictSkip, Deterministic: true},
	}

	cr := scorer.Score(results)

	if !cr.ValidScore {
		t.Fatal("expected valid score with 2 non-skip results")
	}
	// Score should only reflect translation_validation and mutation_testing.
	// Weights: TV=0.30, MT=0.25, total=0.55.
	// Weighted: (0.90*0.30 + 0.85*0.25) / 0.55 = (0.27+0.2125)/0.55 = 0.877...
	expected := (0.90*0.30 + 0.85*0.25) / (0.30 + 0.25)
	if math.Abs(cr.CompositeScore-expected) > 0.001 {
		t.Errorf("expected score %.3f, got %.3f", expected, cr.CompositeScore)
	}
}

func TestScorerInvalidWithFewerThanTwo(t *testing.T) {
	scorer := NewScorer(DefaultScorerConfig())

	tests := []struct {
		name    string
		results []TechniqueResult
	}{
		{"zero results", nil},
		{"one result", []TechniqueResult{
			{Name: "mutation_testing", Score: 0.90, Verdict: VerdictPass, Deterministic: true},
		}},
		{"all skip", []TechniqueResult{
			{Name: "mutation_testing", Score: 0.0, Verdict: VerdictSkip, Deterministic: true},
			{Name: "differential_testing", Score: 0.0, Verdict: VerdictSkip, Deterministic: true},
		}},
		{"one pass one skip", []TechniqueResult{
			{Name: "mutation_testing", Score: 0.90, Verdict: VerdictPass, Deterministic: true},
			{Name: "differential_testing", Score: 0.0, Verdict: VerdictSkip, Deterministic: true},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cr := scorer.Score(tt.results)
			if cr.ValidScore {
				t.Error("expected ValidScore=false")
			}
			if cr.Action != ActionHumanReview {
				t.Errorf("expected human_review when invalid, got %s", cr.Action)
			}
		})
	}
}

func TestDeterministicWeight(t *testing.T) {
	scorer := NewScorer(DefaultScorerConfig())
	results := []TechniqueResult{
		{Name: "translation_validation", Score: 0.90, Verdict: VerdictPass, Deterministic: false},
		{Name: "mutation_testing", Score: 0.85, Verdict: VerdictPass, Deterministic: true},
		{Name: "differential_testing", Score: 0.80, Verdict: VerdictPass, Deterministic: true},
	}

	dw := scorer.DeterministicWeight(results)

	// Deterministic: MT=0.25 + DT=0.20 = 0.45. Total: 0.30+0.25+0.20 = 0.75.
	expected := 0.45 / 0.75
	if math.Abs(dw-expected) > 0.001 {
		t.Errorf("expected deterministic weight %.3f, got %.3f", expected, dw)
	}
}

func TestDeterministicWeightMeetsMinimum(t *testing.T) {
	scorer := NewScorer(DefaultScorerConfig())
	// Default weights: MT=0.25, DT=0.20, PBT=0.15, CI=0.10 are deterministic (total=0.70).
	// TV=0.30 is hybrid. 0.70/(0.70+0.30) = 0.70 >= 0.50.
	results := []TechniqueResult{
		{Name: "translation_validation", Score: 0.80, Verdict: VerdictPass, Deterministic: false},
		{Name: "mutation_testing", Score: 0.80, Verdict: VerdictPass, Deterministic: true},
		{Name: "differential_testing", Score: 0.80, Verdict: VerdictPass, Deterministic: true},
		{Name: "property_based_testing", Score: 0.80, Verdict: VerdictPass, Deterministic: true},
		{Name: "contract_injection", Score: 0.80, Verdict: VerdictPass, Deterministic: true},
	}

	dw := scorer.DeterministicWeight(results)
	if dw < scorer.config.MinDeterministic {
		t.Errorf("deterministic weight %.3f below minimum %.2f", dw, scorer.config.MinDeterministic)
	}
}

func TestScorerCustomWeights(t *testing.T) {
	config := ScorerConfig{
		Weights: map[string]float64{
			"translation_validation": 0.50,
			"mutation_testing":       0.50,
		},
		AcceptThreshold:  0.80,
		MendThreshold:    0.50,
		MinDeterministic: 0.50,
	}
	scorer := NewScorer(config)
	results := []TechniqueResult{
		{Name: "translation_validation", Score: 1.0, Verdict: VerdictPass, Deterministic: false},
		{Name: "mutation_testing", Score: 0.60, Verdict: VerdictFail, Deterministic: true},
	}

	cr := scorer.Score(results)

	expected := (1.0*0.50 + 0.60*0.50) / (0.50 + 0.50)
	if math.Abs(cr.CompositeScore-expected) > 0.001 {
		t.Errorf("expected score %.3f, got %.3f", expected, cr.CompositeScore)
	}
	if cr.Action != ActionAccept {
		t.Errorf("expected accept, got %s", cr.Action)
	}
}

func TestScorerUnknownTechniqueGetsDefaultWeight(t *testing.T) {
	scorer := NewScorer(DefaultScorerConfig())
	results := []TechniqueResult{
		{Name: "translation_validation", Score: 0.90, Verdict: VerdictPass, Deterministic: false},
		{Name: "unknown_technique", Score: 0.80, Verdict: VerdictPass, Deterministic: true},
	}

	cr := scorer.Score(results)

	// TV=0.30, unknown=0.10 (default). Weighted: (0.90*0.30 + 0.80*0.10) / 0.40
	expected := (0.90*0.30 + 0.80*0.10) / (0.30 + 0.10)
	if math.Abs(cr.CompositeScore-expected) > 0.001 {
		t.Errorf("expected score %.3f, got %.3f", expected, cr.CompositeScore)
	}
}
