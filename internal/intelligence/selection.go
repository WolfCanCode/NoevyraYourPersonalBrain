package intelligence

// selectContextCandidates preserves ranking and counts only duplicates visited before the budget cutoff.
func selectContextCandidates(candidates []ContextItem, budget int) ([]ContextItem, int, int) {
	selected := []ContextItem{}
	seen := map[string]bool{}
	used, duplicates := 0, 0
	for _, item := range candidates {
		key := fingerprint(item.Content)
		if seen[key] {
			duplicates++
			continue
		}
		seen[key] = true
		remaining := budget - used
		if remaining < 40 {
			break
		}
		if item.EstimatedTokens > remaining {
			item.Content = truncateTokens(item.Content, remaining)
			item.EstimatedTokens = estimateTokens(item.Content)
		}
		if item.EstimatedTokens == 0 {
			continue
		}
		selected = append(selected, item)
		used += item.EstimatedTokens
	}
	return selected, used, duplicates
}
