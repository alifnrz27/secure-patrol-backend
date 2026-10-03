package dto

import (
	"os"
	"path/filepath"
	"secure-patrol-backend/helper"
	"secure-patrol-backend/pkg/imageutil"
)

// thumbnailSuffix names the cached thumbnail stored next to a scan photo.
const thumbnailSuffix = ".thumb.jpg"

// ScanPhotoThumbnail returns the export thumbnail of a stored scan photo. It is
// made once (decoding a phone photo is the slow part) and kept next to the
// photo, so later exports only read a small file.
func ScanPhotoThumbnail(path string) ([]byte, error) {
	full, err := helper.StoragePath(path)
	if err != nil {
		return nil, err
	}
	if cached, err := os.ReadFile(full + thumbnailSuffix); err == nil && len(cached) > 0 {
		return cached, nil
	}

	original, err := os.ReadFile(full)
	if err != nil {
		return nil, err
	}
	thumb, _, _, err := imageutil.Thumbnail(original, thumbMaxWidth, thumbMaxHeight, thumbQuality)
	if err != nil {
		return nil, err
	}

	// Write to a temporary file first so a concurrent export never reads half a file.
	tmp, err := os.CreateTemp(filepath.Dir(full), ".thumb-*")
	if err == nil {
		_, writeErr := tmp.Write(thumb)
		closeErr := tmp.Close()
		if writeErr == nil && closeErr == nil {
			err = os.Rename(tmp.Name(), full+thumbnailSuffix)
		}
		if writeErr != nil || closeErr != nil || err != nil {
			os.Remove(tmp.Name())
		}
	}
	// A failed cache write only makes the next export slower.
	return thumb, nil
}

// HasScanPhotoThumbnail reports whether the cached thumbnail of a photo exists.
func HasScanPhotoThumbnail(path string) bool {
	full, err := helper.StoragePath(path)
	if err != nil {
		return false
	}
	info, err := os.Stat(full + thumbnailSuffix)
	return err == nil && info.Size() > 0
}
