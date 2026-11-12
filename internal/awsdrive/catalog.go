package awsdrive

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"gdf/internal/drive"
)

// Catalog keeps everything about a user's drive in one DynamoDB table, in the
// user's partition. Each question the application asks is one query on a
// sort-key prefix:
//
//	pk = user#<owner>   sk = #file#<key>            my files
//	pk = user#<owner>   sk = #index#<term>#<key>    files found by a label or alias
//	pk = user#<owner>   sk = #label#<label>         the labels I have
//
// Writes happen once per upload, in the background; reads happen on every
// search. So the pipeline writes one small item per answer, and no read ever
// needs a second lookup or a scan through a growing list.
type Catalog struct {
	client *dynamodb.Client
	table  string
}

// NewCatalog returns a catalog stored in the given table.
func NewCatalog(client *dynamodb.Client, table string) *Catalog {
	return &Catalog{client: client, table: table}
}

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

// RegisterFile writes the record of a new upload, replacing the record of
// any earlier upload of the same key. From then on the record belongs to
// file.Run.
func (c *Catalog) RegisterFile(ctx context.Context, file drive.File) error {
	return c.put(ctx, itemOf(file))
}

// SaveFile writes the whole record, on the condition that file.Run still
// owns it. When a newer upload has been registered in the meantime it writes
// nothing and returns drive.ErrSuperseded.
//
// A second write with the same record changes nothing, so the call is safe
// to retry.
func (c *Catalog) SaveFile(ctx context.Context, file drive.File) error {
	err := c.put(ctx, itemOf(file), func(input *dynamodb.PutItemInput) {
		input.ConditionExpression = aws.String("#run = :run")
		input.ExpressionAttributeNames = map[string]string{"#run": "run"}
		input.ExpressionAttributeValues = map[string]types.AttributeValue{
			":run": &types.AttributeValueMemberS{Value: file.Run},
		}
	})

	var takenOver *types.ConditionalCheckFailedException
	if errors.As(err, &takenOver) {
		return fmt.Errorf("%w: %s", drive.ErrSuperseded, file.Key)
	}

	return err
}

func itemOf(file drive.File) fileItem {
	return fileItem{File: file, PK: userKey(file.Owner), SK: fileKey(file.Key)}
}

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

// File returns the record of one of the owner's uploads.
func (c *Catalog) File(ctx context.Context, owner, key string) (drive.File, error) {
	output, err := c.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(c.table),
		Key: map[string]types.AttributeValue{
			"pk": &types.AttributeValueMemberS{Value: userKey(owner)},
			"sk": &types.AttributeValueMemberS{Value: fileKey(key)},
		},
	})
	if err != nil {
		return drive.File{}, fmt.Errorf("get file record: %w", err)
	}

	if output.Item == nil {
		return drive.File{}, fmt.Errorf("%w: %s", drive.ErrFileNotFound, key)
	}

	var item fileItem
	if err := attributevalue.UnmarshalMapWithOptions(output.Item, &item, useJSONTagsToDecode); err != nil {
		return drive.File{}, fmt.Errorf("decode file record: %w", err)
	}

	return item.File, nil
}

// put writes one item. The conditions, if any, narrow down when it may.
func (c *Catalog) put(ctx context.Context, item any, conditions ...func(*dynamodb.PutItemInput)) error {
	attributes, err := attributevalue.MarshalMapWithOptions(item, useJSONTagsToEncode)
	if err != nil {
		return fmt.Errorf("encode catalog item: %w", err)
	}

	input := &dynamodb.PutItemInput{TableName: aws.String(c.table), Item: attributes}
	for _, condition := range conditions {
		condition(input)
	}

	_, err = c.client.PutItem(ctx, input)
	if err != nil {
		return fmt.Errorf("put catalog item: %w", err)
	}

	return nil
}

// The domain types carry json tags; reuse them for attribute names instead
// of tagging every field a second time.
func useJSONTagsToEncode(options *attributevalue.EncoderOptions) { options.TagKey = "json" }

func useJSONTagsToDecode(options *attributevalue.DecoderOptions) { options.TagKey = "json" }

func userKey(owner string) string { return "user#" + owner }

func fileKey(objectKey string) string { return "#file#" + objectKey }

func labelKey(label string) string { return "#label#" + strings.ToLower(label) }

func indexKey(term, objectKey string) string {
	return "#index#" + strings.ToLower(term) + "#" + objectKey
}
