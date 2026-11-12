package drive

import "fmt"

// ReviewPolicy decides which uploads a person has to look at before they are
// published.
type ReviewPolicy struct {
	// MinConfidence is the confidence, from 0 to 100, that at least one label
	// must reach for the labels to be trusted without review.
	MinConfidence float64
}

// Reasons lists why an upload needs review. An empty list means it can be
// published as it is.
func (p ReviewPolicy) Reasons(analysis Analysis) []string {
	reasons := make([]string, 0, len(analysis.Flags)+1)

	for _, flag := range analysis.Flags {
		reasons = append(reasons, "flagged as "+flag.Name)
	}

	if bestConfidence(analysis.Labels) < p.MinConfidence {
		reasons = append(reasons, fmt.Sprintf("no label reaches %.0f%% confidence", p.MinConfidence))
	}

	return reasons
}

func bestConfidence(labels []Label) float64 {
	best := 0.0
	for _, label := range labels {
		best = max(best, label.Confidence)
	}

	return best
}
