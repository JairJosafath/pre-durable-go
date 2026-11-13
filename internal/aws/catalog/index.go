package catalog

import (
	"context"

	"gdf/internal/drive"
)

// IndexLabel makes the upload findable by the label's name and by each of
// its aliases, and records that the user has this label. A search for an
// alias is then the same single query as a search for the name.
//
// Every item is keyed by what it describes, so writing it again leaves the
// table as it was: the call is safe to retry.
func (c *Catalog) IndexLabel(ctx context.Context, upload drive.Upload, label drive.Label) error {
	for _, term := range label.SearchTerms() {
		entry := indexItem{
			PK:         userKey(upload.Owner),
			SK:         indexKey(term, upload.Key),
			Key:        upload.Key,
			Label:      label.Name,
			Confidence: label.Confidence,
		}
		if err := c.put(ctx, entry); err != nil {
			return err
		}
	}

	return c.put(ctx, labelItem{
		PK:         userKey(upload.Owner),
		SK:         labelKey(label.Name),
		Label:      label.Name,
		Categories: label.Categories,
		Aliases:    label.Aliases,
	})
}
