package usecase

import (
	"bytes"
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok {
			return s
		}
	}
	return ""
}

func getInt(m map[string]interface{}, key string, def int) int {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return int(v)
		case int:
			return v
		case string:
			if i, err := strconv.Atoi(v); err == nil {
				return i
			}
		}
	}
	return def
}

func getFloat(m map[string]interface{}, key string, def float64) float64 {
	if val, ok := m[key]; ok {
		switch v := val.(type) {
		case float64:
			return v
		case int:
			return float64(v)
		case string:
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				return f
			}
		}
	}
	return def
}

func getStringWithFallback(m map[string]interface{}, key string, fallback string) string {
	if val, ok := m[key]; ok {
		if s, ok := val.(string); ok && s != "" {
			return s
		}
	}
	return fallback
}
func convertImageToPDF(fileName string, content []byte) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	ext := strings.ToLower(filepath.Ext(fileName))
	imageType := "JPG"
	if ext == ".png" {
		imageType = "PNG"
	}

	opt := gofpdf.ImageOptions{ImageType: imageType, ReadDpi: true, AllowNegativePosition: true}
	info := pdf.RegisterImageOptionsReader(fileName, opt, bytes.NewReader(content))
	if info == nil {
		return nil, fmt.Errorf("failed to register image with gofpdf")
	}

	pageW, pageH := pdf.GetPageSize()
	maxWidth := pageW - 20
	maxHeight := pageH - 20

	imgW := info.Width()
	imgH := info.Height()

	scale := maxWidth / imgW
	if (maxHeight / imgH) < scale {
		scale = maxHeight / imgH
	}

	finalW := imgW * scale
	finalH := imgH * scale

	x := (pageW - finalW) / 2
	y := (pageH - finalH) / 2

	pdf.ImageOptions(fileName, x, y, finalW, finalH, false, opt, 0, "")

	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
