package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"shoper/db"
	"shoper/middleware"
	"shoper/models"
	"shoper/utils"
)

func adminBrandsHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodPost && !middleware.ValidateCSRF(r) {
		http.Error(w, "csrf token invalid", http.StatusForbidden)
		return
	}
	if r.Method == http.MethodPost {
		deleteID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("delete_id")), 10, 64)
		if deleteID > 0 {
			_, _ = db.DB.Exec(`DELETE FROM brands WHERE id = ?`, deleteID)
			http.Redirect(w, r, "/admin/brands", http.StatusSeeOther)
			return
		}

		id, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("id")), 10, 64)
		name := strings.TrimSpace(r.FormValue("name"))
		logo := strings.TrimSpace(r.FormValue("logo"))
		logoData := strings.TrimSpace(r.FormValue("logo_data"))
		desc := strings.TrimSpace(r.FormValue("description"))
		if name == "" {
			renderAdminBrands(w, r, "请填写品牌名称", "error")
			return
		}
		if id > 0 {
			if logoData != "" {
				if saved, err := utils.SaveBrandLogoData(logoData); err == nil {
					logo = saved
				}
			}
			if db.BrandNameExists(name, id) {
				renderAdminBrands(w, r, "品牌名称已存在", "error")
				return
			}
			_, _ = db.DB.Exec(`UPDATE brands SET name = ?, logo = ?, description = ? WHERE id = ?`, name, logo, desc, id)
		} else {
			if logoData != "" {
				if saved, err := utils.SaveBrandLogoData(logoData); err == nil {
					logo = saved
				}
			}
			if db.BrandNameExists(name, 0) {
				renderAdminBrands(w, r, "品牌名称已存在", "error")
				return
			}
			_, _ = db.DB.Exec(`INSERT INTO brands (name, logo, description, sort_order) VALUES (?, ?, ?, ?)`, name, logo, desc, 0)
		}
		http.Redirect(w, r, "/admin/brands", http.StatusSeeOther)
		return
	}
	renderAdminBrands(w, r, "", "")
}

func renderAdminBrands(w http.ResponseWriter, r *http.Request, message, messageType string) {
	page, size := parsePage(r)
	brands, total, err := db.GetAllBrandsPaged(page, size)
	if err != nil {
		brands, _ = db.GetAllBrands()
		total = len(brands)
	}
	render(w, r, models.PageData{View: "admin", AdminView: "brands", Title: "品牌管理 - Shoper", AllBrands: brands, Message: message, MessageType: messageType, Pagination: buildPagination("/admin/brands", filterQuery(r), page, size, total)})
}
