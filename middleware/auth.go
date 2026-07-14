package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var (
	adminUsername string
	adminPassword string
)

func SetCredentials(username, password string) {
	adminUsername = username
	adminPassword = password
}

func GetUsername() string {
	return adminUsername
}

func GetPassword() string {
	return adminPassword
}

func SetUsername(username string) {
	adminUsername = username
}

func SetPassword(password string) {
	adminPassword = password
}

func SetAdminCookie(w http.ResponseWriter) {
	value := adminUsername + ":" + Sign(adminUsername)
	http.SetCookie(w, &http.Cookie{Name: "admin", Value: value, Path: "/", MaxAge: 86400 * 7, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

func IsAdmin(r *http.Request) bool {
	cookie, err := r.Cookie("admin")
	if err != nil {
		return false
	}
	idx := strings.LastIndex(cookie.Value, ":")
	if idx < 0 {
		return false
	}
	username := cookie.Value[:idx]
	signature := cookie.Value[idx+1:]
	if username != adminUsername {
		return false
	}
	return hmac.Equal([]byte(signature), []byte(Sign(username)))
}

func Sign(value string) string {
	h := hmac.New(sha256.New, []byte(adminPassword))
	h.Write([]byte(value))
	return hex.EncodeToString(h.Sum(nil))
}

func RandomHash() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return time.Now().Format("20060102150405") + hex.EncodeToString(b)
}

// SetCSRFToken 生成 CSRF token 写入 cookie，并返回 token 字符串供模板渲染。
func SetCSRFToken(w http.ResponseWriter) string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	token := hex.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
	return token
}

// GetCSRFToken 从请求 cookie 读取 CSRF token；不存在返回空串。
func GetCSRFToken(r *http.Request) string {
	c, err := r.Cookie("csrf_token")
	if err != nil {
		return ""
	}
	return c.Value
}

// ValidateCSRF 校验表单字段 csrf_token 与 cookie csrf_token 是否一致。
func ValidateCSRF(r *http.Request) bool {
	cookie := GetCSRFToken(r)
	if cookie == "" {
		return false
	}
	form := r.FormValue("csrf_token")
	if form == "" {
		return false
	}
	return hmac.Equal([]byte(cookie), []byte(form))
}

// ClearCSRFToken 清除 CSRF cookie。
func ClearCSRFToken(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     "csrf_token",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: false,
		SameSite: http.SameSiteLaxMode,
	})
}
