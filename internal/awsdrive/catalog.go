package awsdrive

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
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
