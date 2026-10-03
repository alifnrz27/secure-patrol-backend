package facedetect

import (
	"bytes"
	"errors"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"secure-patrol-backend/pkg/imageutil"
	"testing"
)

// testdata/face_frontal.jpg is pigo's sample image (MIT, see testdata/NOTICE).
func sampleFace(t *testing.T) image.Image {
	t.Helper()
	f, err := os.Open("testdata/face_frontal.jpg")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

func encodeJPEG(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func background(width, height int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{235, 235, 235, 255}}, image.Point{}, draw.Src)
	return img
}

// tilt rotates the image around its center by deg degrees.
func tilt(img image.Image, deg float64) image.Image {
	b := img.Bounds()
	out := background(b.Dx(), b.Dy())
	rad := deg * math.Pi / 180
	cx, cy := float64(b.Dx())/2, float64(b.Dy())/2
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			sx := math.Cos(rad)*(float64(x)-cx) + math.Sin(rad)*(float64(y)-cy) + cx
			sy := -math.Sin(rad)*(float64(x)-cx) + math.Cos(rad)*(float64(y)-cy) + cy
			if sx >= 0 && sy >= 0 && int(sx) < b.Dx() && int(sy) < b.Dy() {
				out.Set(x, y, img.At(int(sx), int(sy)))
			}
		}
	}
	return out
}

// storedAs returns the pixels a camera would store for a photo that must be
// displayed with the given EXIF orientation (3, 6 or 8).
func storedAs(img image.Image, orientation int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var out *image.RGBA
	switch orientation {
	case 3:
		out = image.NewRGBA(image.Rect(0, 0, w, h))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out.Set(w-1-x, h-1-y, img.At(x, y))
			}
		}
	case 6: // displayed after a 90° clockwise turn, so stored turned counter-clockwise
		out = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out.Set(y, w-1-x, img.At(x, y))
			}
		}
	case 8: // displayed after a 90° counter-clockwise turn, so stored turned clockwise
		out = image.NewRGBA(image.Rect(0, 0, h, w))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out.Set(h-1-y, x, img.At(x, y))
			}
		}
	}
	return out
}

// withOrientation inserts an EXIF APP1 segment holding the orientation tag.
func withOrientation(jpegData []byte, orientation int, bigEndian bool) []byte {
	var tiff []byte
	if bigEndian {
		tiff = []byte{'M', 'M', 0, 42, 0, 0, 0, 8, 0, 1, 0x01, 0x12, 0, 3, 0, 0, 0, 1, 0, byte(orientation), 0, 0, 0, 0, 0, 0}
	} else {
		tiff = []byte{'I', 'I', 42, 0, 8, 0, 0, 0, 1, 0, 0x12, 0x01, 3, 0, 1, 0, 0, 0, byte(orientation), 0, 0, 0, 0, 0, 0, 0}
	}
	payload := append([]byte("Exif\x00\x00"), tiff...)
	length := len(payload) + 2
	segment := append([]byte{0xFF, 0xE1, byte(length >> 8), byte(length)}, payload...)

	out := append([]byte{}, jpegData[:2]...) // SOI
	out = append(out, segment...)
	return append(out, jpegData[2:]...)
}

func TestValidateAcceptsFrontalFace(t *testing.T) {
	face := sampleFace(t)
	var pngData bytes.Buffer
	png.Encode(&pngData, face)

	for name, data := range map[string][]byte{"jpeg": encodeJPEG(t, face), "png": pngData.Bytes(), "tilted 10°": encodeJPEG(t, tilt(face, 10))} {
		result, err := Validate(data, DefaultOptions())
		if err != nil {
			t.Errorf("%s: expected valid face, got %v (result %+v)", name, err, result)
		}
	}
}

func TestValidateRejects(t *testing.T) {
	face := sampleFace(t)
	b := face.Bounds()

	twoFaces := background(b.Dx()*2, b.Dy())
	draw.Draw(twoFaces, b, face, image.Point{}, draw.Src)
	draw.Draw(twoFaces, b.Add(image.Pt(b.Dx(), 0)), face, image.Point{}, draw.Src)

	smallFace := background(b.Dx()*6, b.Dy()*6)
	draw.Draw(smallFace, b.Add(image.Pt(b.Dx()*3, b.Dy()*3)), face, image.Point{}, draw.Src)

	cases := []struct {
		name string
		data []byte
		want error
	}{
		{"not an image", []byte("definitely not an image"), ErrImageUnreadable},
		{"no face", encodeJPEG(t, background(600, 800)), ErrNoFace},
		{"two faces", encodeJPEG(t, twoFaces), ErrMultipleFaces},
		{"face too small", encodeJPEG(t, smallFace), ErrFaceTooSmall},
		{"head tilted 30°", encodeJPEG(t, tilt(face, 30)), ErrFaceNotFrontal},
	}

	for _, c := range cases {
		result, err := Validate(c.data, DefaultOptions())
		if !errors.Is(err, c.want) {
			t.Errorf("%s: got %v (result %+v), want %v", c.name, err, result, c.want)
		}
	}

	var multiple *MultipleFacesError
	if _, err := Validate(encodeJPEG(t, twoFaces), DefaultOptions()); !errors.As(err, &multiple) || multiple.Count != 2 {
		t.Errorf("two faces: want MultipleFacesError with count 2, got %v", err)
	}
}

func TestValidateUsesExifOrientation(t *testing.T) {
	face := sampleFace(t)

	for _, orientation := range []int{3, 6, 8} {
		stored := encodeJPEG(t, storedAs(face, orientation))

		// Without the EXIF tag the face is upside down or sideways.
		if _, err := Validate(stored, DefaultOptions()); err == nil {
			t.Errorf("orientation %d without EXIF: expected a rejection", orientation)
		}

		for _, bigEndian := range []bool{false, true} {
			if result, err := Validate(withOrientation(stored, orientation, bigEndian), DefaultOptions()); err != nil {
				t.Errorf("orientation %d (big endian %v): expected valid face, got %v (result %+v)", orientation, bigEndian, err, result)
			}
		}
	}
}

func TestJpegOrientation(t *testing.T) {
	base := encodeJPEG(t, background(8, 8))
	if got := imageutil.JPEGOrientation(base); got != 1 {
		t.Errorf("no EXIF: got %d, want 1", got)
	}
	for _, o := range []int{1, 3, 6, 8} {
		if got := imageutil.JPEGOrientation(withOrientation(base, o, false)); got != o {
			t.Errorf("little endian orientation %d: got %d", o, got)
		}
		if got := imageutil.JPEGOrientation(withOrientation(base, o, true)); got != o {
			t.Errorf("big endian orientation %d: got %d", o, got)
		}
	}
	if got := imageutil.JPEGOrientation([]byte{0xFF, 0xD8, 0xFF, 0xE1, 0xFF, 0xFF}); got != 1 {
		t.Errorf("truncated segment: got %d, want 1", got)
	}
}

func TestOrientRotatesBuffer(t *testing.T) {
	// 2 rows x 3 cols: 1 2 3 / 4 5 6
	pixels := []uint8{1, 2, 3, 4, 5, 6}
	cases := map[int]struct {
		want       []uint8
		rows, cols int
	}{
		3: {[]uint8{6, 5, 4, 3, 2, 1}, 2, 3},
		6: {[]uint8{4, 1, 5, 2, 6, 3}, 3, 2},
		8: {[]uint8{3, 6, 2, 5, 1, 4}, 3, 2},
	}
	for orientation, c := range cases {
		got, rows, cols := orient(pixels, 2, 3, orientation)
		if rows != c.rows || cols != c.cols || !bytes.Equal(got, c.want) {
			t.Errorf("orientation %d: got %v (%dx%d), want %v (%dx%d)", orientation, got, rows, cols, c.want, c.rows, c.cols)
		}
	}
}
