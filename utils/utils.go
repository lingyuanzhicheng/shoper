package utils

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FormatPrice(cents int64) string {
	return fmt.Sprintf("%.2f", float64(cents)/100)
}

func FirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func Must(err error) {
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		panic(err)
	}
}

func ValueAt(values []string, i int) string {
	if i < 0 || i >= len(values) {
		return ""
	}
	return values[i]
}

func SplitUniqueTags(input string) ([]string, string) {
	parts := strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == '，' || r == '\n' || r == '、'
	})
	seen := make(map[string]bool)
	tags := make([]string, 0, len(parts))
	for _, part := range parts {
		tag := strings.TrimSpace(part)
		if tag == "" {
			continue
		}
		key := strings.ToLower(tag)
		if seen[key] {
			return nil, tag
		}
		seen[key] = true
		tags = append(tags, tag)
	}
	return tags, ""
}

func SaveBrandLogoData(dataURL string) (string, error) {
	return SaveImageDataURL(dataURL)
}

func SaveImageDataURL(dataURL string) (string, error) {
	comma := strings.Index(dataURL, ",")
	if comma >= 0 {
		dataURL = dataURL[comma+1:]
	}
	raw, err := base64.StdEncoding.DecodeString(dataURL)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	name := hex.EncodeToString(sum[:]) + ".jpg"
	if err := os.MkdirAll("uploads", 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join("uploads", name), raw, 0644); err != nil {
		return "", err
	}
	return "/uploads/" + name, nil
}
