package main

import (
	"context"
	"errors"
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	ddbtypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"

	"gdf/internal/aws/catalog"
	"gdf/internal/aws/objects"
	"gdf/internal/drive"
)

// services are the local stand-ins the workflow runs against.
type services struct {
	s3      *s3.Client
	dynamo  *dynamodb.Client
	objects *objects.Objects
	catalog *catalog.Catalog
}

const localRegion = "us-east-1"

// localCredentials are placeholders: the local containers accept any keys.
func localCredentials() aws.Credentials {
	return aws.Credentials{AccessKeyID: "local", SecretAccessKey: "local"}
}

// connect builds the AWS SDK clients, pointed at the local containers. The
// credentials are placeholders: nothing here ever talks to AWS.
func connect(opts options) *services {
	keys := localCredentials()
	local := aws.Config{
		Region:      localRegion,
		Credentials: credentials.NewStaticCredentialsProvider(keys.AccessKeyID, keys.SecretAccessKey, ""),
	}

	storage := s3.NewFromConfig(local, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(opts.s3URL)
		o.UsePathStyle = true
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})

	database := dynamodb.NewFromConfig(local, func(o *dynamodb.Options) {
		o.BaseEndpoint = aws.String(opts.dynamoURL)
	})

	return &services{
		s3:      storage,
		dynamo:  database,
		objects: objects.New(storage),
		catalog: catalog.New(database, tableName),
	}
}

// prepare creates the buckets and the table. In AWS, CDK does this.
func (s *services) prepare(ctx context.Context) error {
	for _, bucket := range []string{uploadBucket, thumbnailBucket} {
		if err := s.createBucket(ctx, bucket); err != nil {
			return err
		}
	}

	return s.createTable(ctx)
}

func (s *services) createBucket(ctx context.Context, bucket string) error {
	_, err := s.s3.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(bucket)})

	var (
		owned  *s3types.BucketAlreadyOwnedByYou
		exists *s3types.BucketAlreadyExists
	)

	if err != nil && !errors.As(err, &owned) && !errors.As(err, &exists) {
		return fmt.Errorf("create bucket %s (is the local stack up? run `make local-up`): %w", bucket, err)
	}

	return nil
}

func (s *services) createTable(ctx context.Context) error {
	_, err := s.dynamo.CreateTable(ctx, &dynamodb.CreateTableInput{
		TableName:   aws.String(tableName),
		BillingMode: ddbtypes.BillingModePayPerRequest,
		AttributeDefinitions: []ddbtypes.AttributeDefinition{
			{AttributeName: aws.String("pk"), AttributeType: ddbtypes.ScalarAttributeTypeS},
			{AttributeName: aws.String("sk"), AttributeType: ddbtypes.ScalarAttributeTypeS},
		},
		KeySchema: []ddbtypes.KeySchemaElement{
			{AttributeName: aws.String("pk"), KeyType: ddbtypes.KeyTypeHash},
			{AttributeName: aws.String("sk"), KeyType: ddbtypes.KeyTypeRange},
		},
	})

	var exists *ddbtypes.ResourceInUseException
	if err != nil && !errors.As(err, &exists) {
		return fmt.Errorf("create table %s: %w", tableName, err)
	}

	return nil
}

// upload puts the image into the upload bucket, as the browser does with a
// presigned URL in AWS.
func (s *services) upload(ctx context.Context, owner, path string) (drive.Upload, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return drive.Upload{}, fmt.Errorf("read image: %w", err)
	}

	upload, err := drive.ParseUpload(uploadBucket, drive.UserKey(owner, objectName(path)))
	if err != nil {
		return drive.Upload{}, err
	}

	contentType := mime.TypeByExtension(filepath.Ext(path))
	if err := s.objects.Put(ctx, upload.Object, data, contentType); err != nil {
		return drive.Upload{}, err
	}

	fmt.Printf("Uploaded %s to s3://%s/%s\n", path, upload.Bucket, upload.Key)

	return upload, nil
}
