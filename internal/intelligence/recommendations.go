package intelligence

import (
	"math"
	"sort"
	"strings"

	"github.com/WolfCanCode/personal-ai-brain/internal/memory"
)

func capabilityRecommendations(store *memory.Store, task string, profile ProjectProfile) ([]Recommendation, error) {
	all := []string{task}
	all = append(all, profile.Languages...)
	all = append(all, profile.Stacks...)
	all = append(all, profile.Signals...)
	wanted := terms(all...)
	recommendations := []Recommendation{}
	seen := map[string]bool{}
	for _, item := range ProvisionCatalog {
		hits := []string{}
		for _, trigger := range item.TriggerSignals {
			if wanted[trigger] {
				hits = append(hits, trigger)
			}
		}
		if len(hits) == 0 {
			continue
		}
		confidence := math.Round(math.Min(0.95, 0.45+float64(len(hits))*0.12)*100) / 100
		recommendations = append(recommendations, Recommendation{item.Kind, item.Ref, item.Reason + " Signals: " + strings.Join(hits, ", ") + ".", confidence, false})
		seen[item.Kind+":"+item.Ref] = true
	}
	approved, err := store.Skills()
	if err != nil {
		return nil, err
	}
	for _, skill := range approved {
		keyword := strings.ToLower(strings.TrimSpace(skill.Keyword))
		if keyword == "" || (!wanted[keyword] && !strings.Contains(strings.ToLower(task), keyword)) {
			continue
		}
		key := "skill:" + skill.Ref
		if seen[key] {
			continue
		}
		recommendations = append(recommendations, Recommendation{"skill", skill.Ref, "Previously reviewed skill matches keyword " + keyword + ".", 0.95, true})
	}
	sort.SliceStable(recommendations, func(i, j int) bool { return recommendations[i].Confidence > recommendations[j].Confidence })
	if len(recommendations) > 6 {
		recommendations = recommendations[:6]
	}
	return recommendations, nil
}

func reasoningFrame(task string, profile ProjectProfile) ReasoningFrame {
	values := []string{task}
	values = append(values, profile.Languages...)
	values = append(values, profile.Stacks...)
	values = append(values, profile.Signals...)
	wanted := terms(values...)
	frame := ReasoningFrame{
		Approach: []string{
			"Inspect the smallest authoritative project evidence before choosing a change.",
			"Reuse existing conventions and define the minimal safe change boundary.",
			"Implement in reversible increments and preserve unrelated user work.",
		},
		Risks: []string{"Stale external guidance or memory may conflict with the current repository."},
		VerificationGates: []string{
			"Run the narrowest relevant checks, then the repository's required validation.",
			"Verify the exact artifact or user-visible path, not only helper functions.",
			"Record the outcome and any durable decision for the next agent.",
		},
	}
	if wanted["security"] || wanted["auth"] || wanted["payment"] || wanted["webhook"] {
		frame.Approach = append(frame.Approach, "Map trust boundaries and attacker-controlled inputs before implementation.")
		frame.Risks = append(frame.Risks, "Authentication, authorization, replay, secret handling and unsafe input paths require explicit checks.")
		frame.VerificationGates = append(frame.VerificationGates, "Exercise at least one negative or adversarial case.")
	}
	if wanted["database"] || wanted["sql"] || wanted["migration"] || wanted["sqlite"] || wanted["postgres"] {
		frame.Approach = append(frame.Approach, "Treat schema compatibility, transaction boundaries and rollback as part of the feature.")
		frame.Risks = append(frame.Risks, "Partial migrations, concurrent writers and irreversible data changes can invalidate a locally correct implementation.")
		frame.VerificationGates = append(frame.VerificationGates, "Test upgrade from the previous schema and reopen the resulting store.")
	}
	if wanted["api"] || wanted["mcp"] || wanted["contract"] {
		frame.Approach = append(frame.Approach, "Write the compatibility contract and error behavior before changing the adapter.")
		frame.Risks = append(frame.Risks, "Protocol drift and silently accepted invalid input create downstream agent failures.")
		frame.VerificationGates = append(frame.VerificationGates, "Test valid, invalid and backward-compatible protocol paths.")
	}
	if wanted["frontend"] || wanted["ui"] || wanted["web"] || wanted["browser"] {
		frame.Approach = append(frame.Approach, "Trace the user-visible flow across data, interaction, loading, error and empty states.")
		frame.VerificationGates = append(frame.VerificationGates, "Verify the final browser state at the intended viewport and interaction path.")
	}
	if wanted["release"] || wanted["deployment"] || wanted["ci-cd"] {
		frame.Risks = append(frame.Risks, "A source-level pass is insufficient if packaging, permissions or the deployed artifact differs.")
		frame.VerificationGates = append(frame.VerificationGates, "Validate checksums, packaged executables and publication state.")
	}
	return frame
}
