// Package awsdrive implements PrimeDrive's services on AWS: S3 for objects,
// DynamoDB for the catalog, Rekognition for vision and the Lambda API for
// review callbacks and execution histories.
//
// The S3 and DynamoDB adapters only need an endpoint that speaks the same
// API, so the local stack reuses them against SeaweedFS and DynamoDB Local.
//
// One file per job:
//
//	objects.go        S3: read, write and delete objects
//	objects_urls.go   S3: presigned upload and download URLs
//	catalog.go        DynamoDB: the table layout, and how an item is written
//	catalog_files.go  DynamoDB: the file records
//	catalog_index.go  DynamoDB: the label index
//	catalog_items.go  DynamoDB: the items and their keys
//	vision.go         Rekognition: labels and moderation concerns
//	callbacks.go      Lambda: review decisions for suspended executions
//	executions.go     Lambda: what a durable execution has done so far
//
// Nothing else is in the package.
package awsdrive
