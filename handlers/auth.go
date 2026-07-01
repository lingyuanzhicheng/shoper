package handlers

import (
	"net/http"
	"strings"

	"shoper/middleware"
	"shoper/models"
)

func loginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		phone := strings.TrimSpace(r.FormValue("phone"))
		key := strings.TrimSpace(r.FormValue("key"))
		if phone == middleware.GetPhone() && key == middleware.GetSecret() {
			middleware.SetAdminCookie(w)
			http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
			return
		}
		render(w, r, models.PageData{View: "login", Title: "后台登录 - Shoper", Message: "电话号码或密钥不正确", MessageType: "error"})
		return
	}
	render(w, r, models.PageData{View: "login", Title: "后台登录 - Shoper"})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "admin", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
