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

func adminCategoriesHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodPost {
		name := strings.TrimSpace(r.FormValue("name"))
		if name != "" {
			if db.CategoryNameExists(name) {
				renderAdminCategories(w, r, "分类名称已存在，请使用其他名称", "error")
				return
			}

			initialTags, dupTag := utils.SplitUniqueTags(r.FormValue("initial_tags"))
			if dupTag != "" {
				renderAdminCategories(w, r, "同一分类下标签不能重复：「"+dupTag+"」", "error")
				return
			}

			res, err := db.DB.Exec(`INSERT INTO categories (name, sort_order) VALUES (?, ?)`, name, 0)
			if err == nil {
				categoryID, _ := res.LastInsertId()
				for _, tag := range initialTags {
					_, _ = db.DB.Exec(`INSERT INTO tags (name, category_id, sort_order) VALUES (?, ?, ?)`, tag, categoryID, 0)
				}
			}
		}

		tagName := strings.TrimSpace(r.FormValue("tag_name"))
		tagCatStr := strings.TrimSpace(r.FormValue("tag_category_id"))
		if tagName != "" && tagCatStr != "" {
			tagCatID, _ := strconv.ParseInt(tagCatStr, 10, 64)
			if tagCatID > 0 {
				if db.TagNameExistsInCategory(tagCatID, tagName) {
					renderAdminCategories(w, r, "该分类下已存在标签：「"+tagName+"」", "error")
					return
				}
				_, _ = db.DB.Exec(`INSERT INTO tags (name, category_id, sort_order) VALUES (?, ?, ?)`, tagName, tagCatID, 0)
			}
		}

		delStr := strings.TrimSpace(r.FormValue("delete_id"))
		if delStr != "" {
			delID, _ := strconv.ParseInt(delStr, 10, 64)
			db.DB.Exec(`UPDATE products SET group_name = '' WHERE group_name = (SELECT name FROM categories WHERE id = ?)`, delID)
			db.DB.Exec(`DELETE FROM tags WHERE category_id = ?`, delID)
			db.DB.Exec(`DELETE FROM categories WHERE id = ?`, delID)
		}

		delTagStr := strings.TrimSpace(r.FormValue("delete_tag_id"))
		if delTagStr != "" {
			delTagID, _ := strconv.ParseInt(delTagStr, 10, 64)
			db.DB.Exec(`DELETE FROM tags WHERE id = ?`, delTagID)
		}

		http.Redirect(w, r, "/admin/categories", http.StatusSeeOther)
		return
	}
	renderAdminCategories(w, r, "", "")
}

func renderAdminCategories(w http.ResponseWriter, r *http.Request, message, messageType string) {
	page, size := parsePage(r)
	cats, total, err := db.GetAllCategoriesPaged(page, size)
	if err != nil {
		cats, _ = db.GetAllCategories()
		total = len(cats)
	}
	for i := range cats {
		cats[i].Tags, _ = db.GetTagsByCategory(cats[i].ID)
	}
	allCats, _ := db.GetAllCategories()
	render(w, r, models.PageData{View: "admin", AdminView: "categories", Title: "分类管理 - Shoper", AllCategories: allCats, ParentCategories: cats, Message: message, MessageType: messageType, Pagination: buildPagination("/admin/categories", filterQuery(r), page, size, total)})
}
