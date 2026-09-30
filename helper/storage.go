package helper

import (
	"encoding/base64"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const MaxFacePhotoSize = 5 * 1024 * 1024 // 5 MB

var (
	ErrFileTooLarge       = errors.New("file size must not exceed 5 MB")
	ErrFileTypeNotAllowed = errors.New("file must be a JPEG or PNG image")
)

var allowedImageTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
}

// StorageRoot is the private directory for uploaded files. It is never served statically.
func StorageRoot() string {
	if root := os.Getenv("STORAGE_PATH"); root != "" {
		return root
	}
	return "./storage"
}

// ReadImage reads an uploaded image into memory after checking its size and its
// type by content (not by extension or client supplied content type). It returns
// the bytes and the file extension to store it with.
func ReadImage(file *multipart.FileHeader) ([]byte, string, error) {
	if file.Size > MaxFacePhotoSize {
		return nil, "", ErrFileTooLarge
	}

	src, err := file.Open()
	if err != nil {
		return nil, "", err
	}
	defer src.Close()

	data, err := io.ReadAll(io.LimitReader(src, MaxFacePhotoSize+1))
	if err != nil {
		return nil, "", err
	}
	if len(data) > MaxFacePhotoSize {
		return nil, "", ErrFileTooLarge
	}

	ext, ok := allowedImageTypes[http.DetectContentType(data)]
	if !ok {
		return nil, "", ErrFileTypeNotAllowed
	}

	return data, ext, nil
}

// SaveImageBytes stores image bytes under StorageRoot()/dir with a random name
// and returns the path relative to StorageRoot().
func SaveImageBytes(data []byte, ext string, dir string) (string, error) {
	name, err := RandomHex(16)
	if err != nil {
		return "", err
	}

	relativePath := filepath.Join(dir, name+ext)
	fullPath := filepath.Join(StorageRoot(), relativePath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o750); err != nil {
		return "", err
	}

	dst, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o640)
	if err != nil {
		return "", err
	}

	if _, err := dst.Write(data); err != nil {
		dst.Close()
		os.Remove(fullPath)
		return "", err
	}
	if err := dst.Close(); err != nil {
		os.Remove(fullPath)
		return "", err
	}

	return relativePath, nil
}

// SaveImage validates an uploaded image and stores it (see ReadImage and SaveImageBytes).
func SaveImage(file *multipart.FileHeader, dir string) (string, error) {
	data, ext, err := ReadImage(file)
	if err != nil {
		return "", err
	}
	return SaveImageBytes(data, ext, dir)
}

// StoragePath resolves a stored relative path, refusing paths that escape StorageRoot().
func StoragePath(relativePath string) (string, error) {
	root, err := filepath.Abs(StorageRoot())
	if err != nil {
		return "", err
	}

	full := filepath.Join(root, filepath.Clean("/"+relativePath))
	if !strings.HasPrefix(full, root+string(os.PathSeparator)) {
		return "", errors.New("invalid storage path")
	}

	return full, nil
}

func DeleteStoredFile(relativePath string) {
	if relativePath == "" {
		return
	}
	if full, err := StoragePath(relativePath); err == nil {
		os.Remove(full)
	}
}

// ReadImageBase64 reads a stored image and returns it base64 encoded together
// with its detected content type.
func ReadImageBase64(relativePath string) (encoded string, contentType string, err error) {
	fullPath, err := StoragePath(relativePath)
	if err != nil {
		return "", "", err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", "", err
	}

	return base64.StdEncoding.EncodeToString(content), http.DetectContentType(content), nil
}
