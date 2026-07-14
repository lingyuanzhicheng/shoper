package handlers

import (
	"net/http"
	"strings"

	"shoper/middleware"
	"shoper/models"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		username := strings.TrimSpace(r.FormValue("username"))
		password := strings.TrimSpace(r.FormValue("password"))
		if username == middleware.GetUsername() && password == middleware.GetPassword() {
			middleware.SetAdminCookie(w)
			middleware.SetCSRFToken(w)
			http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
			return
		}
		token := middleware.SetCSRFToken(w)
		render(w, r, models.PageData{View: "login", Title: "后台登录 - Shoper", Message: "用户名或密码不正确", MessageType: "error", CSRFToken: token})
		return
	}
	token := middleware.SetCSRFToken(w)
	render(w, r, models.PageData{View: "login", Title: "后台登录 - Shoper", CSRFToken: token})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "admin", Value: "", Path: "/", MaxAge: -1})
	middleware.ClearCSRFToken(w)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
