package imagenet_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"gdf/internal/drive"
	"gdf/internal/drive/drivetest"
	"gdf/internal/local/imagenet"
)

const (
	classes       = 1001 // background + 1000 ImageNet classes, like the ResNet-50 model
	tabbyIndex    = 282
	beerGlassRow  = 442
	inputSide     = 224
	colorChannels = 3
)

// modelServer imitates TensorFlow Serving: it checks the shape of the image
// it is sent and answers with fixed probabilities.
func modelServer(t *testing.T, probabilities map[int]float64) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var body struct {
			Instances [][][][]float32 `json:"instances"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
			t.Errorf("request body: %v", err)
		}

		if len(body.Instances) != 1 || len(body.Instances[0]) != inputSide ||
			len(body.Instances[0][0]) != inputSide || len(body.Instances[0][0][0]) != colorChannels {
			t.Errorf("the model expects one %dx%dx%d image", inputSide, inputSide, colorChannels)
		}

		scores := make([]float64, classes)
		for index, probability := range probabilities {
			scores[index] = probability
		}

		if err := json.NewEncoder(writer).Encode(map[string]any{"predictions": [][]float64{scores}}); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
	t.Cleanup(server.Close)

	return server
}

func visionFor(t *testing.T, probabilities map[int]float64) (*imagenet.Vision, drive.Object) {
	t.Helper()

	photo := drive.Object{Bucket: "uploads", Key: "users/alice/photo.png"}
	objects := drivetest.NewObjects()
	objects.Add(photo, drivetest.PNG(t, 640, 480))

	server := modelServer(t, probabilities)

	return imagenet.NewVision(objects, server.Client(), server.URL, "resnet"), photo
}

func TestDetectLabelsReturnsTheMostLikelyClasses(t *testing.T) {
	t.Parallel()

	vision, photo := visionFor(t, map[int]float64{tabbyIndex: 0.82, tabbyIndex + 1: 0.11, 7: 0.01})

	labels, err := vision.DetectLabels(t.Context(), photo)
	if err != nil {
		t.Fatalf("DetectLabels: %v", err)
	}

	// "tiger cat" can also be found as "cat"; "tabby" has no more general word in its name.
	want := []drive.Label{{Name: "tabby", Confidence: 82}, {Name: "tiger cat", Confidence: 11, Aliases: []string{"cat"}}}
	if !reflect.DeepEqual(labels, want) {
		t.Errorf("labels = %+v, want %+v", labels, want)
	}
}

func TestDetectModerationLabelsMapsClassesToCategories(t *testing.T) {
	t.Parallel()

	vision, photo := visionFor(t, map[int]float64{beerGlassRow: 0.64, beerGlassRow - 1: 0.2})

	flags, err := vision.DetectModerationLabels(t.Context(), photo)
	if err != nil {
		t.Fatalf("DetectModerationLabels: %v", err)
	}

	// Beer glass and beer bottle are both alcohol: one flag, with the higher confidence.
	if want := []drive.Label{{Name: "Alcohol", Confidence: 64}}; !reflect.DeepEqual(flags, want) {
		t.Errorf("flags = %+v, want one Alcohol flag at 64%%", flags)
	}
}
