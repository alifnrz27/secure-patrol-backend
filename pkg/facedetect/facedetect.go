// Package facedetect checks that a photo is usable as a reference face photo:
// exactly one face, large enough, looking at the camera with both eyes visible.
// It uses pigo (pure Go, no cgo) with its cascades embedded in the binary.
package facedetect

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"sort"
	"sync"

	pigo "github.com/esimov/pigo/core"
)

var (
	//go:embed cascade/facefinder
	facefinderCascade []byte

	//go:embed cascade/puploc
	puplocCascade []byte
)

var (
	ErrImageUnreadable = errors.New("face photo could not be read as an image")
	ErrNoFace          = errors.New("no face detected in the photo, use a clear and well lit photo of the face")
	ErrMultipleFaces   = errors.New("the photo must contain exactly one face")
	ErrFaceTooSmall    = errors.New("the face is too small, take the photo closer so the face fills more of the picture")
	ErrFaceNotFrontal  = errors.New("the face must look straight at the camera with both eyes visible")
)

// MultipleFacesError reports how many faces were found. It matches ErrMultipleFaces with errors.Is.
type MultipleFacesError struct {
	Count int
}

func (e *MultipleFacesError) Error() string {
	return fmt.Sprintf("%s, %d faces were found", ErrMultipleFaces.Error(), e.Count)
}

func (e *MultipleFacesError) Unwrap() error { return ErrMultipleFaces }

// Options are the acceptance thresholds.
type Options struct {
	// MinFaceRatio is the minimum face width relative to the shorter side of the photo.
	MinFaceRatio float64
	// MaxTiltDegrees is the maximum angle of the line between the eyes.
	MaxTiltDegrees float64
	// MaxTurnRatio is the maximum horizontal offset of the eyes' midpoint from the face
	// center, relative to the face width (how far the head may be turned).
	MaxTurnRatio float64
}

// DefaultOptions are the built-in thresholds. The application passes the
// values stored in the database settings instead.
func DefaultOptions() Options {
	return Options{MinFaceRatio: 0.2, MaxTiltDegrees: 20, MaxTurnRatio: 0.12}
}

// Result describes the detected face, for logging and tuning.
type Result struct {
	Faces       int
	FaceRatio   float64
	TiltDegrees float64
	TurnRatio   float64
	EyesFound   bool
}

const (
	// Photos are scaled down before detection; faces we accept are far larger than this needs.
	maxDetectionSide = 800
	// Minimum pigo score for a detection to count as a face.
	minQuality = 5.0
)

var (
	loadOnce     sync.Once
	loadErr      error
	faceCascade  *pigo.Pigo
	pupilCascade *pigo.PuplocCascade
)

func load() error {
	loadOnce.Do(func() {
		faceCascade, loadErr = pigo.NewPigo().Unpack(facefinderCascade)
		if loadErr != nil {
			return
		}
		pupilCascade, loadErr = pigo.NewPuplocCascade().UnpackCascade(puplocCascade)
	})
	return loadErr
}

// Validate returns nil when the image is usable as a reference face photo.
func Validate(data []byte, opts Options) (Result, error) {
	var result Result

	if err := load(); err != nil {
		return result, err
	}

	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return result, ErrImageUnreadable
	}

	pixels, rows, cols := grayscale(img, maxDetectionSide)
	if format == "jpeg" {
		pixels, rows, cols = orient(pixels, rows, cols, jpegOrientation(data))
	}
	params := pigo.ImageParams{Pixels: pixels, Rows: rows, Cols: cols, Dim: cols}

	shortSide := rows
	if cols < shortSide {
		shortSide = cols
	}

	detections := faceCascade.RunCascade(pigo.CascadeParams{
		// Small enough to also notice other people in the background.
		MinSize:     maxInt(20, shortSide/20),
		MaxSize:     shortSide,
		ShiftFactor: 0.1,
		ScaleFactor: 1.1,
		ImageParams: params,
	}, 0)
	detections = faceCascade.ClusterDetections(detections, 0.2)

	var candidates []pigo.Detection
	for _, detection := range detections {
		if detection.Q >= minQuality {
			candidates = append(candidates, detection)
		}
	}
	faces := mergeOverlapping(candidates)

	result.Faces = len(faces)
	switch {
	case len(faces) == 0:
		return result, ErrNoFace
	case len(faces) > 1:
		return result, &MultipleFacesError{Count: len(faces)}
	}

	face := faces[0]
	result.FaceRatio = float64(face.Scale) / float64(shortSide)
	if result.FaceRatio < opts.MinFaceRatio {
		return result, ErrFaceTooSmall
	}

	left := findPupil(face, params, -1)
	right := findPupil(face, params, 1)
	result.EyesFound = left != nil && right != nil
	if !result.EyesFound {
		return result, ErrFaceNotFrontal
	}

	result.TiltDegrees = math.Abs(math.Atan2(float64(right.Row-left.Row), float64(right.Col-left.Col)) * 180 / math.Pi)
	result.TurnRatio = math.Abs(float64(left.Col+right.Col)/2-float64(face.Col)) / float64(face.Scale)
	if result.TiltDegrees > opts.MaxTiltDegrees || result.TurnRatio > opts.MaxTurnRatio {
		return result, ErrFaceNotFrontal
	}

	return result, nil
}

// mergeOverlapping keeps one detection per face. pigo's clustering can return
// the same face twice at slightly different scales, which would otherwise be
// counted as two faces.
func mergeOverlapping(detections []pigo.Detection) []pigo.Detection {
	sort.Slice(detections, func(i, j int) bool { return detections[i].Q > detections[j].Q })

	var kept []pigo.Detection
	for _, detection := range detections {
		duplicate := false
		for _, face := range kept {
			if overlap(detection, face) > 0.3 {
				duplicate = true
				break
			}
		}
		if !duplicate {
			kept = append(kept, detection)
		}
	}
	return kept
}

// overlap is the intersection over union of two square detections.
func overlap(a, b pigo.Detection) float64 {
	box := func(d pigo.Detection) (float64, float64, float64, float64) {
		half := float64(d.Scale) / 2
		return float64(d.Col) - half, float64(d.Row) - half, float64(d.Col) + half, float64(d.Row) + half
	}
	ax1, ay1, ax2, ay2 := box(a)
	bx1, by1, bx2, by2 := box(b)

	width := math.Min(ax2, bx2) - math.Max(ax1, bx1)
	height := math.Min(ay2, by2) - math.Max(ay1, by1)
	if width <= 0 || height <= 0 {
		return 0
	}
	intersection := width * height
	union := float64(a.Scale*a.Scale+b.Scale*b.Scale) - intersection
	return intersection / union
}

// findPupil looks for the pupil on one side of the face (side -1 = left, 1 = right),
// using the eye positions suggested by pigo's pupil localization example.
func findPupil(face pigo.Detection, params pigo.ImageParams, side int) *pigo.Puploc {
	pupil := pupilCascade.RunDetector(pigo.Puploc{
		Row:      face.Row - int(0.085*float32(face.Scale)),
		Col:      face.Col + side*int(0.185*float32(face.Scale)),
		Scale:    float32(face.Scale) * 0.4,
		Perturbs: 50,
	}, params, 0, false)
	if pupil == nil || pupil.Row <= 0 || pupil.Col <= 0 {
		return nil
	}
	return pupil
}

// grayscale converts the image to 8-bit luminance, scaled down with a box filter
// so the longer side is at most maxSide. JPEG luminance is read directly from
// the Y plane, which avoids a slow per-pixel color conversion.
func grayscale(img image.Image, maxSide int) ([]uint8, int, int) {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	step := 1
	for width/step > maxSide || height/step > maxSide {
		step++
	}
	cols, rows := width/step, height/step
	pixels := make([]uint8, rows*cols)

	ycbcr, isYCbCr := img.(*image.YCbCr)

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			var sum, count int
			for dy := 0; dy < step; dy++ {
				y := bounds.Min.Y + r*step + dy
				for dx := 0; dx < step; dx++ {
					x := bounds.Min.X + c*step + dx
					if isYCbCr {
						sum += int(ycbcr.Y[ycbcr.YOffset(x, y)])
					} else {
						red, green, blue, _ := img.At(x, y).RGBA()
						// ITU-R BT.601 luma, on 16-bit channels.
						sum += int((19595*red + 38470*green + 7471*blue + 1<<15) >> 24)
					}
					count++
				}
			}
			pixels[r*cols+c] = uint8(sum / count)
		}
	}

	return pixels, rows, cols
}

// orient applies an EXIF orientation (1-8) to a grayscale buffer, so photos
// taken with a rotated phone are checked the way they are displayed.
func orient(pixels []uint8, rows, cols, orientation int) ([]uint8, int, int) {
	if orientation < 2 || orientation > 8 {
		return pixels, rows, cols
	}

	outRows, outCols := rows, cols
	if orientation >= 5 {
		outRows, outCols = cols, rows
	}
	out := make([]uint8, len(pixels))

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			var nr, nc int
			switch orientation {
			case 2: // mirrored horizontally
				nr, nc = r, cols-1-c
			case 3: // rotated 180
				nr, nc = rows-1-r, cols-1-c
			case 4: // mirrored vertically
				nr, nc = rows-1-r, c
			case 5: // transposed
				nr, nc = c, r
			case 6: // rotated 90 clockwise
				nr, nc = c, rows-1-r
			case 7: // transversed
				nr, nc = cols-1-c, rows-1-r
			case 8: // rotated 90 counter-clockwise
				nr, nc = cols-1-c, r
			}
			out[nr*outCols+nc] = pixels[r*cols+c]
		}
	}

	return out, outRows, outCols
}

// jpegOrientation reads the EXIF orientation tag (0x0112) of a JPEG; 1 when absent.
func jpegOrientation(data []byte) int {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 1
	}

	for i := 2; i+4 <= len(data); {
		if data[i] != 0xFF {
			return 1
		}
		marker := data[i+1]
		if marker == 0xDA || marker == 0xD9 { // start of scan / end of image
			return 1
		}
		length := int(data[i+2])<<8 | int(data[i+3])
		if length < 2 || i+2+length > len(data) {
			return 1
		}
		segment := data[i+4 : i+2+length]
		if marker == 0xE1 && len(segment) > 6 && string(segment[:6]) == "Exif\x00\x00" {
			return tiffOrientation(segment[6:])
		}
		i += 2 + length
	}
	return 1
}

func tiffOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}

	var u16 func([]byte) uint16
	var u32 func([]byte) uint32
	switch string(tiff[:2]) {
	case "II":
		u16 = func(b []byte) uint16 { return uint16(b[0]) | uint16(b[1])<<8 }
		u32 = func(b []byte) uint32 { return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24 }
	case "MM":
		u16 = func(b []byte) uint16 { return uint16(b[1]) | uint16(b[0])<<8 }
		u32 = func(b []byte) uint32 { return uint32(b[3]) | uint32(b[2])<<8 | uint32(b[1])<<16 | uint32(b[0])<<24 }
	default:
		return 1
	}

	offset := int(u32(tiff[4:8]))
	if offset < 8 || offset+2 > len(tiff) {
		return 1
	}
	entries := int(u16(tiff[offset : offset+2]))
	for e := 0; e < entries; e++ {
		entry := offset + 2 + e*12
		if entry+12 > len(tiff) {
			return 1
		}
		if u16(tiff[entry:entry+2]) == 0x0112 {
			value := int(u16(tiff[entry+8 : entry+10]))
			if value >= 1 && value <= 8 {
				return value
			}
			return 1
		}
	}
	return 1
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
