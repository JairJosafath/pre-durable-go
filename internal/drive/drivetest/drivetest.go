// Package drivetest provides in-memory implementations of the services
// PrimeDrive depends on. They make the workflow testable on a laptop with no
// network, no credentials and no containers.
package drivetest

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/png"
	"slices"
	"sync"
	"testing"

	"gdf/internal/drive"
)

// ErrNoSuchObject reports a read of an object that is not in the store.
var ErrNoSuchObject = errors.New("no such object")

// ErrUnavailable is the transient failure Vision reports while it is flaky.
var ErrUnavailable = errors.New("vision service unavailable")

// Vision answers every image with the same scripted labels.
type Vision struct {
	Labels []drive.Label
	Flags  []drive.Label
	// Err, when set, fails every call.
	Err error
	// OnDetect, when set, runs at the start of every label detection. A test
	// uses it to make something happen while the pipeline is analysing.
	OnDetect func()

	mu          sync.Mutex
	failuresDue int
	calls       int
}

// FailFirst makes the next n label detections fail with ErrUnavailable.
func (v *Vision) FailFirst(n int) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.failuresDue = n
}

// Calls reports how often labels were requested, failed attempts included.
func (v *Vision) Calls() int {
	v.mu.Lock()
	defer v.mu.Unlock()

	return v.calls
}

// DetectLabels returns the scripted labels.
func (v *Vision) DetectLabels(context.Context, drive.Object) ([]drive.Label, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.calls++

	if v.OnDetect != nil {
		v.OnDetect()
	}

	if v.failuresDue > 0 {
		v.failuresDue--

		return nil, ErrUnavailable
	}

	return v.Labels, v.Err
}

// DetectModerationLabels returns the scripted moderation flags.
func (v *Vision) DetectModerationLabels(context.Context, drive.Object) ([]drive.Label, error) {
	return v.Flags, v.Err
}

// Catalog keeps file records and label index entries in maps.
type Catalog struct {
	mu     sync.Mutex
	files  map[string]drive.File
	index  map[string][]string
	labels map[string]drive.Label
}

// NewCatalog returns an empty catalog.
func NewCatalog() *Catalog {
	return &Catalog{
		files:  map[string]drive.File{},
		index:  map[string][]string{},
		labels: map[string]drive.Label{},
	}
}

// RegisterFile stores the record of a new upload, replacing any earlier one
// for the same key. The record then belongs to file.Run.
func (c *Catalog) RegisterFile(_ context.Context, file drive.File) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.files[file.Key] = file

	return nil
}

// SaveFile replaces the record while file.Run still owns it, like the real
// catalog. After a newer upload was registered it returns drive.ErrSuperseded.
func (c *Catalog) SaveFile(_ context.Context, file drive.File) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.files[file.Key].Run != file.Run {
		return fmt.Errorf("%w: %s", drive.ErrSuperseded, file.Key)
	}

	c.files[file.Key] = file

	return nil
}

// IndexLabel makes the upload findable by the label's name and aliases, and
// remembers the label. Indexing twice leaves single entries, like the real
// catalog.
func (c *Catalog) IndexLabel(_ context.Context, upload drive.Upload, label drive.Label) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, term := range label.SearchTerms() {
		if !slices.Contains(c.index[term], upload.Key) {
			c.index[term] = append(c.index[term], upload.Key)
		}
	}

	c.labels[label.Name] = label

	return nil
}

// Label returns what the catalog knows about a label the user has.
func (c *Catalog) Label(name string) (drive.Label, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	label, ok := c.labels[name]

	return label, ok
}

// File returns the stored record for an object key.
func (c *Catalog) File(key string) (drive.File, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	file, ok := c.files[key]

	return file, ok
}

// Keys returns the object keys found by a search term.
func (c *Catalog) Keys(term string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()

	return slices.Clone(c.index[term])
}

// Objects is an object store backed by a map.
type Objects struct {
	mu    sync.Mutex
	blobs map[drive.Object][]byte
}

// NewObjects returns an empty object store.
func NewObjects() *Objects {
	return &Objects{blobs: map[drive.Object][]byte{}}
}

// Add puts an object into the store.
func (o *Objects) Add(object drive.Object, data []byte) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.blobs[object] = data
}

// Has reports whether the object is in the store.
func (o *Objects) Has(object drive.Object) bool {
	o.mu.Lock()
	defer o.mu.Unlock()

	_, ok := o.blobs[object]

	return ok
}

// Get returns the object's content.
func (o *Objects) Get(_ context.Context, object drive.Object) ([]byte, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	data, ok := o.blobs[object]
	if !ok {
		return nil, fmt.Errorf("%w: %s/%s", ErrNoSuchObject, object.Bucket, object.Key)
	}

	return data, nil
}

// Put stores the object.
func (o *Objects) Put(_ context.Context, object drive.Object, data []byte, _ string) error {
	o.Add(object, data)

	return nil
}

// Delete removes the object. Deleting a missing object is not an error.
func (o *Objects) Delete(_ context.Context, object drive.Object) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	delete(o.blobs, object)

	return nil
}

// PNG returns an encoded blank PNG of the given size.
func PNG(tb testing.TB, width, height int) []byte {
	tb.Helper()

	var buffer bytes.Buffer
	if err := png.Encode(&buffer, image.NewRGBA(image.Rect(0, 0, width, height))); err != nil {
		tb.Fatalf("encode test image: %v", err)
	}

	return buffer.Bytes()
}
