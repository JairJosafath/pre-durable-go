package awsdrive

import (
	"context"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"gdf/internal/drive"
)

// Catalog keeps file records and label index entries in one DynamoDB table.
//
//	pk = user#<owner>   sk = #file#<key>            one item per upload
//	pk = user#<owner>   sk = #index#<label>#<key>   one item per detected label
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

// indexItem makes one upload findable by one label.
type indexItem struct {
	PK         string  `json:"pk"`
	SK         string  `json:"sk"`
	Key        string  `json:"key"`
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
}

// SaveFile writes the whole record. A second write with the same record
// changes nothing, so the call is safe to retry.
func (c *Catalog) SaveFile(ctx context.Context, file drive.File) error {
	return c.put(ctx, fileItem{File: file, PK: userKey(file.Owner), SK: fileKey(file.Key)})
}

// IndexLabel makes the upload findable by the label. The item's key is the
// label and the object together, so indexing twice leaves a single item.
func (c *Catalog) IndexLabel(ctx context.Context, upload drive.Upload, label drive.Label) error {
	return c.put(ctx, indexItem{
		PK:         userKey(upload.Owner),
		SK:         indexKey(label.Name, upload.Key),
		Key:        upload.Key,
		Label:      label.Name,
		Confidence: label.Confidence,
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

func (c *Catalog) put(ctx context.Context, item any) error {
	attributes, err := attributevalue.MarshalMapWithOptions(item, useJSONTagsToEncode)
	if err != nil {
		return fmt.Errorf("encode catalog item: %w", err)
	}

	_, err = c.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(c.table), Item: attributes})
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

func indexKey(label, objectKey string) string {
	return "#index#" + strings.ToLower(label) + "#" + objectKey
}
