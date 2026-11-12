// Package pipeline is the PrimeDrive upload workflow, written as a durable
// function. It replaces the Step Functions state machine of the original
// project: the same stages, expressed as ordinary Go.
//
// The workflow only talks to the outside world through the small interfaces
// declared in services.go. Production wires in AWS services, the tests wire
// in in-memory fakes, and the local runner wires in open-source stand-ins.
//
// Start reading in pipeline.go: Handle and process are the whole story. Each
// stage they call lives in the file named after it, in this order:
//
//	event.go     read the upload out of the S3 event
//	register.go  claim the upload for this run
//	analysis.go  label the image and make its thumbnail, side by side
//	review.go    ask a person, when the analysis gives a reason to
//	publish.go   make an approved upload searchable
//	discard.go   remove a rejected upload
//
// The rest supports those stages:
//
//	record.go    how a stage writes the catalog record
//	endings.go   a run that failed, or that a newer upload overtook
//	retry.go     which failures are worth another attempt
//	services.go  the services the workflow needs
//	config.go    the settings that differ between environments
//
// Nothing else is in the package.
package pipeline
