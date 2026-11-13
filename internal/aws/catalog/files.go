package catalog

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"

	"gdf/internal/drive"
)

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
