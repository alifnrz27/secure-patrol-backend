// Package imageutil holds small image helpers shared by face detection and exports.
package imageutil

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
)

// Thumbnail decodes a JPEG or PNG, applies its EXIF orientation (so phone
// photos are upright) and scales it down to fit maxWidth x maxHeight. It
// returns the thumbnail as JPEG and its size. Smaller images are not enlarged.
func Thumbnail(data []byte, maxWidth, maxHeight int, quality int) ([]byte, int, int, error) {
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, err
	}
	if format == "jpeg" {
		img = Orient(img, JPEGOrientation(data))
	}

	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return nil, 0, 0, image.ErrFormat
	}

	scale := 1.0
	if sx := float64(maxWidth) / float64(width); sx < scale {
		scale = sx
	}
	if sy := float64(maxHeight) / float64(height); sy < scale {
		scale = sy
	}
	outWidth, outHeight := maxIntValue(1, int(float64(width)*scale)), maxIntValue(1, int(float64(height)*scale))

	out := downscale(img, outWidth, outHeight)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, out, &jpeg.Options{Quality: quality}); err != nil {
		return nil, 0, 0, err
	}
	return buf.Bytes(), outWidth, outHeight, nil
}

// downscale averages a few source pixels for each output pixel (box filter),
// which gives clean thumbnails without an extra dependency. JPEG (YCbCr) and
// RGBA images are read directly, which is many times faster than image.At.
func downscale(src image.Image, width, height int) *image.RGBA {
	bounds := src.Bounds()
	srcWidth, srcHeight := bounds.Dx(), bounds.Dy()
	out := image.NewRGBA(image.Rect(0, 0, width, height))

	var sample func(x, y int) (r, g, b uint32)
	switch img := src.(type) {
	case *image.YCbCr:
		sample = func(x, y int) (uint32, uint32, uint32) {
			yy := img.Y[img.YOffset(x, y)]
			c := img.COffset(x, y)
			r, g, b := color.YCbCrToRGB(yy, img.Cb[c], img.Cr[c])
			return uint32(r), uint32(g), uint32(b)
		}
	case *image.RGBA:
		sample = func(x, y int) (uint32, uint32, uint32) {
			i := img.PixOffset(x, y)
			return uint32(img.Pix[i]), uint32(img.Pix[i+1]), uint32(img.Pix[i+2])
		}
	case *image.NRGBA:
		sample = func(x, y int) (uint32, uint32, uint32) {
			i := img.PixOffset(x, y)
			return uint32(img.Pix[i]), uint32(img.Pix[i+1]), uint32(img.Pix[i+2])
		}
	default:
		sample = func(x, y int) (uint32, uint32, uint32) {
			r, g, b, _ := img.At(x, y).RGBA()
			return r >> 8, g >> 8, b >> 8
		}
	}

	for y := 0; y < height; y++ {
		y0 := bounds.Min.Y + y*srcHeight/height
		y1 := bounds.Min.Y + (y+1)*srcHeight/height
		if y1 <= y0 {
			y1 = y0 + 1
		}
		for x := 0; x < width; x++ {
			x0 := bounds.Min.X + x*srcWidth/width
			x1 := bounds.Min.X + (x+1)*srcWidth/width
			if x1 <= x0 {
				x1 = x0 + 1
			}

			// Sample at most 3x3 points per output pixel to keep large photos fast.
			stepX, stepY := maxIntValue(1, (x1-x0+2)/3), maxIntValue(1, (y1-y0+2)/3)
			var r, g, b, n uint32
			for sy := y0; sy < y1; sy += stepY {
				for sx := x0; sx < x1; sx += stepX {
					pr, pg, pb := sample(sx, sy)
					r, g, b, n = r+pr, g+pg, b+pb, n+1
				}
			}
			i := out.PixOffset(x, y)
			out.Pix[i], out.Pix[i+1], out.Pix[i+2], out.Pix[i+3] = uint8(r/n), uint8(g/n), uint8(b/n), 255
		}
	}
	return out
}

// Orient applies an EXIF orientation (1-8) to an image.
func Orient(img image.Image, orientation int) image.Image {
	if orientation < 2 || orientation > 8 {
		return img
	}
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	outWidth, outHeight := width, height
	if orientation >= 5 {
		outWidth, outHeight = height, width
	}
	out := image.NewRGBA(image.Rect(0, 0, outWidth, outHeight))

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var nx, ny int
			switch orientation {
			case 2: // mirrored horizontally
				nx, ny = width-1-x, y
			case 3: // rotated 180
				nx, ny = width-1-x, height-1-y
			case 4: // mirrored vertically
				nx, ny = x, height-1-y
			case 5: // transposed
				nx, ny = y, x
			case 6: // rotated 90 clockwise
				nx, ny = height-1-y, x
			case 7: // transversed
				nx, ny = height-1-y, width-1-x
			case 8: // rotated 90 counter-clockwise
				nx, ny = y, width-1-x
			}
			out.Set(nx, ny, img.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return out
}

func maxIntValue(a, b int) int {
	if a > b {
		return a
	}
	return b
}
