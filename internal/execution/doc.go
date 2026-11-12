// Package execution describes what a durable execution has done so far: the
// operations it ran, in which invocations, and how it ended. It is the
// workflow as the Lambda durable execution service recorded it, in a shape a
// web page can draw.
//
// Lambda keeps a history of events for every durable execution: a step
// started, a step failed and will be retried, a callback was opened, an
// invocation ended. The local runner records the same events. FromEvents
// folds such a list into one History.
//
//	history.go    the History and what it is made of
//	fold.go       FromEvents: one event after the other
//	operation.go  the events of steps, contexts, waits, callbacks and invokes
//	failure.go    the error an event carries
//
// Nothing else is in the package.
package execution
