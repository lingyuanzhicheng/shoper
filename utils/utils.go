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
	"strconv"
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
	return SaveImageDataURL(dataURL, "brand")
}

func SaveImageDataURL(dataURL, kind string) (string, error) {
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
	dir := filepath.Join("uploads", kind)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0644); err != nil {
		return "", err
	}
	return "/uploads/" + kind + "/" + name, nil
}

// ParseYuanToCents 将价格字符串直接解析为整数分，不经过 float 中间表示。
// 支持整数（"123"→12300）、1-2 位小数（"123.4"→12340、"123.45"→12345）、负数（"-12.34"→-1234）、空串（→0）。
// 非法格式返回 (0, error)。
func ParseYuanToCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	neg := false
	if s[0] == '-' {
		neg = true
		s = s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}
	intPart := s
	fracPart := ""
	if dot := strings.Index(s, "."); dot >= 0 {
		intPart = s[:dot]
		fracPart = s[dot+1:]
		// 拒绝多个小数点
		if strings.Contains(fracPart, ".") {
			return 0, fmt.Errorf("invalid price format: %q", s)
		}
	}
	// 整数部分允许为空（".45" 视为 0.45）
	var intCents int64
	if intPart != "" {
		v, err := strconv.ParseInt(intPart, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid price format: %q", s)
		}
		intCents = v * 100
	}
	// 小数部分：取前 2 位，不足补 0，超长截断
	var fracCents int64
	if fracPart != "" {
		// 校验小数部分均为数字
		for _, c := range fracPart {
			if c < '0' || c > '9' {
				return 0, fmt.Errorf("invalid price format: %q", s)
			}
		}
		if len(fracPart) >= 2 {
			fracPart = fracPart[:2]
		} else if len(fracPart) == 1 {
			fracPart = fracPart + "0"
		}
		v, err := strconv.ParseInt(fracPart, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid price format: %q", s)
		}
		fracCents = v
	}
	result := intCents + fracCents
	if neg {
		result = -result
	}
	return result, nil
}
