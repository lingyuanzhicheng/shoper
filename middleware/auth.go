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
	adminPhone  string
	adminSecret string
)

func SetCredentials(phone, secret string) {
	adminPhone = phone
	adminSecret = secret
}

func GetPhone() string {
	return adminPhone
}

func GetSecret() string {
	return adminSecret
}

func SetPhone(phone string) {
	adminPhone = phone
}

func SetSecret(secret string) {
	adminSecret = secret
}

func SetAdminCookie(w http.ResponseWriter) {
	value := adminPhone + ":" + Sign(adminPhone)
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
	phone := cookie.Value[:idx]
	signature := cookie.Value[idx+1:]
	if phone != adminPhone {
		return false
	}
	return hmac.Equal([]byte(signature), []byte(Sign(phone)))
}

func Sign(value string) string {
	h := hmac.New(sha256.New, []byte(adminSecret))
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
