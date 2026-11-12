package pipeline_test

import (
	"slices"
	"testing"
)

func TestUploadsAreFoundByLabelAliases(t *testing.T) {
	t.Parallel()

	h := newHarness(t, aCat())
	succeededWith(t, h.run(t, catPhoto))

	// The label itself is catalogued once, with its aliases, so the labels a
	// user has can be listed without reading every indexed file.
	if cat, ok := h.catalog.Label("Cat"); !ok || !slices.Equal(cat.Aliases, []string{"Kitty"}) {
		t.Errorf("label Cat = %+v, want it catalogued with its alias", cat)
	}

	// The photo is indexed under the alias too: searching "Kitty" is the same
	// single lookup as searching "Cat".
	if keys := h.catalog.Keys("Kitty"); !slices.Equal(keys, []string{catPhoto}) {
		t.Errorf("index[Kitty] = %v, want the cat photo", keys)
	}
}
