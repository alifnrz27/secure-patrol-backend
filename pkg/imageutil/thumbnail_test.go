package imageutil

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

func encode(t *testing.T, img image.Image, asPNG bool) []byte {
	t.Helper()
	var buf bytes.Buffer
	var err error
	if asPNG {
		err = png.Encode(&buf, img)
	} else {
		err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	}
	if err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

// A red left half and a blue right half must stay red and blue after resizing.
func TestThumbnailKeepsColorsAndSize(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 1600, 1200))
	for y := 0; y < 1200; y++ {
		for x := 0; x < 1600; x++ {
			c := color.RGBA{220, 20, 20, 255}
			if x >= 800 {
				c = color.RGBA{20, 20, 220, 255}
			}
			src.Set(x, y, c)
		}
	}
	for _, asPNG := range []bool{false, true} {
		data, w, h, err := Thumbnail(encode(t, src, asPNG), 320, 240, 80)
		if err != nil || w != 320 || h != 240 {
			t.Fatalf("png=%v: %dx%d err %v", asPNG, w, h, err)
		}
		thumb, _ := jpeg.Decode(bytes.NewReader(data))
		left, _, leftBlue, _ := thumb.At(40, 120).RGBA()
		_, _, right, _ := thumb.At(280, 120).RGBA()
		if left>>8 < 180 || leftBlue>>8 > 70 || right>>8 < 180 {
			t.Errorf("png=%v: colors changed: left r=%d b=%d, right b=%d", asPNG, left>>8, leftBlue>>8, right>>8)
		}
	}
}

func TestThumbnailDoesNotEnlarge(t *testing.T) {
	data, w, h, err := Thumbnail(encode(t, image.NewRGBA(image.Rect(0, 0, 100, 50)), true), 320, 240, 80)
	if err != nil || w != 100 || h != 50 || len(data) == 0 {
		t.Fatalf("%dx%d err %v", w, h, err)
	}
}
