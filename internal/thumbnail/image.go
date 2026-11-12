package thumbnail

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/draw"
)

const jpegQuality = 85

// Shrink scales an image down so that its longest side is at most maxSide
// pixels. Images that are already small enough are returned unchanged.
func Shrink(source image.Image, maxSide int) image.Image {
	size := source.Bounds().Size()

	longest := max(size.X, size.Y)
	if longest <= maxSide {
		return source
	}

	width := max(1, size.X*maxSide/longest)
	height := max(1, size.Y*maxSide/longest)

	target := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(target, target.Bounds(), source, source.Bounds(), draw.Over, nil)

	return target
}

func encode(img image.Image, format string) ([]byte, string, error) {
	var buffer bytes.Buffer

	if format == "png" {
		if err := png.Encode(&buffer, img); err != nil {
			return nil, "", fmt.Errorf("encode png: %w", err)
		}

		return buffer.Bytes(), "image/png", nil
	}

	if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, "", fmt.Errorf("encode jpeg: %w", err)
	}

	return buffer.Bytes(), "image/jpeg", nil
}
