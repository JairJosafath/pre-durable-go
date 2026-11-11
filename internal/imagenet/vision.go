// Package imagenet recognises image content with an open ImageNet classifier
// (ResNet-50) served by TensorFlow Serving. It is the local, open-source
// stand-in for Amazon Rekognition: the same interface, no cloud.
package imagenet

import (
	"bytes"
	"context"
	_ "embed" // labels.txt is compiled into the binary
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // decode JPEG uploads
	_ "image/png"  // decode PNG uploads
	"net/http"
	"slices"
	"strings"

	"golang.org/x/image/draw"

	"gdf/internal/drive"
)

const (
	inputSide = 224 // the model looks at 224x224 pixels
	channels  = 3
	maxColor  = 0xffff

	topLabels          = 5
	minLabelPercent    = 5
	minConcernPercent  = 15
	percent            = 100
	backgroundClassRow = 1 // labels.txt starts with a "background" row

	alcohol = "Alcohol"
	weapons = "Weapons"
)

// ErrClassifier reports an unusable answer from the model server.
var ErrClassifier = errors.New("classifier")

//go:embed labels.txt
var labelsFile string

// classNames returns the 1000 ImageNet class names, in model output order.
func classNames() []string {
	return strings.Split(strings.TrimSpace(labelsFile), "\n")[backgroundClassRow:]
}

// concerns maps ImageNet classes to moderation categories. Rekognition has a
// model for this; an ImageNet classifier does not, so the few classes that
// matter for the demo are listed by hand.
func concerns() map[string]string {
	return map[string]string{
		"beer glass":      alcohol,
		"beer bottle":     alcohol,
		"wine bottle":     alcohol,
		"red wine":        alcohol,
		"cocktail shaker": alcohol,
		"revolver":        weapons,
		"rifle":           weapons,
		"assault rifle":   weapons,
	}
}

// Images reads the image to classify.
type Images interface {
	Get(ctx context.Context, object drive.Object) ([]byte, error)
}

// Vision classifies images with a model behind TensorFlow Serving's REST API.
type Vision struct {
	images     Images
	client     *http.Client
	predictURL string
}

// NewVision returns a vision service for the model named model on the
// TensorFlow Serving instance at baseURL, such as http://localhost:8501.
func NewVision(images Images, client *http.Client, baseURL, model string) *Vision {
	return &Vision{
		images:     images,
		client:     client,
		predictURL: fmt.Sprintf("%s/v1/models/%s:predict", strings.TrimRight(baseURL, "/"), model),
	}
}

// DetectLabels returns the classes the model finds most likely.
func (v *Vision) DetectLabels(ctx context.Context, object drive.Object) ([]drive.Label, error) {
	labels, err := v.classify(ctx, object)
	if err != nil {
		return nil, err
	}

	labels = slices.DeleteFunc(labels, func(label drive.Label) bool { return label.Confidence < minLabelPercent })

	return labels[:min(topLabels, len(labels))], nil
}

// DetectModerationLabels returns the moderation categories of the classes the
// model recognises, one label per category.
func (v *Vision) DetectModerationLabels(ctx context.Context, object drive.Object) ([]drive.Label, error) {
	labels, err := v.classify(ctx, object)
	if err != nil {
		return nil, err
	}

	categories := concerns()
	flags := make([]drive.Label, 0, 1)

	// Labels are sorted by confidence, so the first hit per category is its strongest.
	for _, label := range labels {
		category, concerning := categories[label.Name]
		alreadyFlagged := slices.ContainsFunc(flags, func(flag drive.Label) bool { return flag.Name == category })

		if concerning && !alreadyFlagged && label.Confidence >= minConcernPercent {
			flags = append(flags, drive.Label{Name: category, Confidence: label.Confidence})
		}
	}

	return flags, nil
}

// classify returns every class with its confidence, most likely first.
func (v *Vision) classify(ctx context.Context, object drive.Object) ([]drive.Label, error) {
	data, err := v.images.Get(ctx, object)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}

	picture, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", drive.ErrUnsupportedImage, err)
	}

	scores, err := v.predict(ctx, tensor(picture))
	if err != nil {
		return nil, err
	}

	return rank(scores)
}

// tensor scales the picture to the model's input size and returns it as
// height x width x channels values between 0 and 1.
func tensor(picture image.Image) [][][]float32 {
	scaled := image.NewRGBA(image.Rect(0, 0, inputSide, inputSide))
	draw.BiLinear.Scale(scaled, scaled.Bounds(), picture, picture.Bounds(), draw.Src, nil)

	rows := make([][][]float32, inputSide)
	for y := range rows {
		rows[y] = make([][]float32, inputSide)
		for x := range rows[y] {
			red, green, blue, _ := scaled.At(x, y).RGBA()
			rows[y][x] = []float32{float32(red) / maxColor, float32(green) / maxColor, float32(blue) / maxColor}
		}
	}

	return rows
}

// predict sends one image to TensorFlow Serving and returns the model's
// score for each class.
func (v *Vision) predict(ctx context.Context, input [][][]float32) ([]float64, error) {
	body, err := json.Marshal(map[string]any{"instances": [][][][]float32{input}})
	if err != nil {
		return nil, fmt.Errorf("encode prediction request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, v.predictURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build prediction request: %w", err)
	}

	response, err := v.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("call model server: %w", err)
	}

	defer response.Body.Close() //nolint:errcheck // nothing to do about a failed close after a read

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: model server answered %s", ErrClassifier, response.Status)
	}

	var answer struct {
		Predictions [][]float64 `json:"predictions"`
	}
	if err := json.NewDecoder(response.Body).Decode(&answer); err != nil {
		return nil, fmt.Errorf("decode prediction: %w", err)
	}

	if len(answer.Predictions) != 1 {
		return nil, fmt.Errorf("%w: expected 1 prediction, got %d", ErrClassifier, len(answer.Predictions))
	}

	return answer.Predictions[0], nil
}

// rank turns the model's scores into labels, most likely first. The model
// outputs one probability per class, with or without a leading background class.
func rank(scores []float64) ([]drive.Label, error) {
	names := classNames()

	offset := len(scores) - len(names)
	if offset != 0 && offset != backgroundClassRow {
		return nil, fmt.Errorf("%w: got %d scores for %d classes", ErrClassifier, len(scores), len(names))
	}

	labels := make([]drive.Label, len(names))
	for index, name := range names {
		labels[index] = drive.Label{Name: name, Confidence: scores[index+offset] * percent, Aliases: generalNames(name)}
	}

	slices.SortFunc(labels, func(a, b drive.Label) int {
		switch {
		case a.Confidence > b.Confidence:
			return -1
		case a.Confidence < b.Confidence:
			return 1
		default:
			return strings.Compare(a.Name, b.Name)
		}
	})

	return labels, nil
}

// generalNames returns the everyday word for a class, to search by. ImageNet
// classes are specific: a cat is a "Siamese cat", a car a "passenger car".
// Rekognition answers "Cat" and adds aliases; this classifier knows no such
// thing, so the last word of a class name stands in for one.
func generalNames(class string) []string {
	words := strings.Fields(class)
	if len(words) < 2 { //nolint:mnd // a single word is already as general as the name gets
		return nil
	}

	return []string{words[len(words)-1]}
}
