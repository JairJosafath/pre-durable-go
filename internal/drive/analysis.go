package drive

// Analysis is everything the vision service found in an image.
type Analysis struct {
	// Labels describe what is in the image.
	Labels []Label `json:"labels"`
	// Flags are moderation concerns, such as alcohol or weapons.
	Flags []Label `json:"flags"`
}

// Label is something recognised in an image, with a confidence from 0 to 100.
type Label struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	// Categories group related labels, such as "Animals and Pets".
	Categories []string `json:"categories,omitempty"`
	// Aliases are other names for the same thing, such as "Kitty" for "Cat".
	Aliases []string `json:"aliases,omitempty"`
}

// SearchTerms returns every word an upload with this label can be found by:
// the label's name and its aliases.
func (l Label) SearchTerms() []string {
	return append([]string{l.Name}, l.Aliases...)
}
