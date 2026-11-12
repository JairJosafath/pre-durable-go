// Package drive holds the PrimeDrive domain: what a user uploaded, what was
// found in it, and when a person has to look at it. It knows nothing about
// AWS or about durable execution.
//
// The files follow an upload through its life:
//
//	upload.go    an object a user put into their drive
//	analysis.go  what was found in the image
//	policy.go    whether a person has to look at it
//	review.go    the request to a reviewer, and the verdict
//	file.go      the catalog record that all of this ends up in
//	status.go    the statuses that record moves through
//	errors.go    what can go wrong
//
// Nothing else is in the package.
package drive
