package awsdrive

import (
	"strings"

	"gdf/internal/drive"
)

// fileItem is a file record with its table keys.
type fileItem struct {
	drive.File

	PK string `json:"pk"`
	SK string `json:"sk"`
}

// labelItem describes a label the user has, once, however many files carry it.
type labelItem struct {
	PK         string   `json:"pk"`
	SK         string   `json:"sk"`
	Label      string   `json:"label"`
	Categories []string `json:"categories,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
}

// indexItem makes one upload findable by one search term: a label's name or
// one of its aliases. Label always holds the name.
type indexItem struct {
	PK         string  `json:"pk"`
	SK         string  `json:"sk"`
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

func userKey(owner string) string { return "user#" + owner }

func fileKey(objectKey string) string { return "#file#" + objectKey }

func labelKey(label string) string { return "#label#" + strings.ToLower(label) }

func indexKey(term, objectKey string) string {
	return "#index#" + strings.ToLower(term) + "#" + objectKey
}
