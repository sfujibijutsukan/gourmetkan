package util

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const DefaultMaxUploadBytes int64 = 5 << 20
const DefaultMaxUploadFiles = 8

// SaveUploadedImage stores an uploaded image from multipart/form-data and returns a web path.
// If no file is selected, it returns an empty path and nil error.
func SaveUploadedImage(r *http.Request, fieldName, outputDir string, maxBytes int64) (string, error) {
	paths, err := SaveUploadedImages(r, fieldName, outputDir, maxBytes, 1)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", nil
	}
	return paths[0], nil
}

// SaveUploadedImages stores multiple uploaded images from multipart/form-data.
// If no files are selected, it returns an empty slice and nil error.
func SaveUploadedImages(r *http.Request, fieldName, outputDir string, maxBytes int64, maxFiles int) ([]string, error) {
	if r.MultipartForm == nil {
		return nil, fmt.Errorf("multipart form is not parsed")
	}
	files := r.MultipartForm.File[fieldName]
	if len(files) == 0 {
		return nil, nil
	}
	if maxFiles <= 0 {
		maxFiles = DefaultMaxUploadFiles
	}
	if len(files) > maxFiles {
		return nil, fmt.Errorf("too many files (max %d)", maxFiles)
	}
	if maxBytes <= 0 {
		maxBytes = DefaultMaxUploadBytes
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create upload dir: %w", err)
	}

	paths := make([]string, 0, len(files))
	for _, header := range files {
		if header == nil || strings.TrimSpace(header.Filename) == "" {
			continue
		}
		webPath, err := saveUploadedFile(header, outputDir, maxBytes)
		if err != nil {
			_ = DeleteUploadedImages(paths)
			return nil, err
		}
		paths = append(paths, webPath)
	}
	return paths, nil
}

// DeleteUploadedImage removes an uploaded image under /static/uploads if it exists.
func DeleteUploadedImage(webPath string) error {
	if strings.TrimSpace(webPath) == "" {
		return nil
	}
	normalized := filepath.ToSlash(filepath.Clean("/" + strings.TrimSpace(webPath)))
	if !strings.HasPrefix(normalized, "/static/uploads/") {
		return nil
	}
	diskPath := strings.TrimPrefix(normalized, "/")
	if err := os.Remove(diskPath); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("remove file: %w", err)
	}
	return nil
}

// DeleteUploadedImages removes multiple uploaded image files and ignores missing files.
func DeleteUploadedImages(webPaths []string) error {
	for _, path := range webPaths {
		if err := DeleteUploadedImage(path); err != nil {
			return err
		}
	}
	return nil
}

func saveUploadedFile(header *multipart.FileHeader, outputDir string, maxBytes int64) (string, error) {
	file, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("read upload: %w", err)
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}
	if int64(len(data)) > maxBytes {
		return "", fmt.Errorf("file is too large (max %d bytes)", maxBytes)
	}

	contentType := http.DetectContentType(data)
	ext, ok := allowedImageExt(contentType)
	if !ok {
		return "", fmt.Errorf("unsupported image type")
	}

	token, err := RandomToken(18)
	if err != nil {
		return "", fmt.Errorf("generate filename: %w", err)
	}
	base := strings.TrimSuffix(filepath.Base(header.Filename), filepath.Ext(header.Filename))
	if base == "" {
		base = "image"
	}
	name := sanitizeFilePart(base) + "_" + token + ext
	path := filepath.Join(outputDir, name)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("save file: %w", err)
	}

	return "/" + filepath.ToSlash(path), nil
}

func allowedImageExt(contentType string) (string, bool) {
	switch contentType {
	case "image/jpeg":
		return ".jpg", true
	case "image/png":
		return ".png", true
	case "image/gif":
		return ".gif", true
	case "image/webp":
		return ".webp", true
	default:
		return "", false
	}
}

func sanitizeFilePart(name string) string {
	name = strings.ToLower(name)
	var b strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "image"
	}
	return b.String()
}
