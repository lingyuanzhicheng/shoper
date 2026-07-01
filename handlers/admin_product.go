package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"shoper/db"
	"shoper/middleware"
	"shoper/models"
)

func adminProductsHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	catID, _ := strconv.ParseInt(r.URL.Query().Get("cat"), 10, 64)
	tagID, _ := strconv.ParseInt(r.URL.Query().Get("tag"), 10, 64)
	group := ""
	catName := ""
	var tags []models.Tag
	if catID > 0 {
		if cat, err := db.GetCategoryByID(catID); err == nil {
			group = cat.Name
			catName = cat.Name
		}
		tags, _ = db.GetTagsByCategory(catID)
	}
	page, size := parsePage(r)
	products, total, err := db.ListProductsPaged("", group, "", page, size)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	cats, _ := db.GetAllCategories()
	for i := range cats {
		cats[i].Tags, _ = db.GetTagsByCategory(cats[i].ID)
	}
	brands, _ := db.GetAllBrands()
	render(w, r, models.PageData{View: "admin", AdminView: "products", Title: "商品管理 - Shoper", AllProducts: products, AllCategories: cats, AllBrands: brands, CategoryID: catID, CategoryName: catName, Tags: tags, TagID: tagID, Pagination: buildPagination("/admin/products", filterQuery(r), page, size, total)})
}

func adminProductNewHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodPost {
		slug := strings.TrimSpace(r.FormValue("slug"))
		name := strings.TrimSpace(r.FormValue("name"))
		brand := strings.TrimSpace(r.FormValue("brand"))
		brandLogo := strings.TrimSpace(r.FormValue("brand_logo"))
		unit := strings.TrimSpace(r.FormValue("unit"))
		priceYuan, _ := strconv.ParseFloat(strings.TrimSpace(r.FormValue("price")), 64)
		desc := strings.TrimSpace(r.FormValue("description"))
		body := strings.TrimSpace(r.FormValue("body"))
		catID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
		if slug == "" {
			slug = fmt.Sprintf("p-%d", time.Now().UnixNano())
		}
		if name == "" || brand == "" || unit == "" || priceYuan <= 0 {
			cats, _ := db.GetAllCategories()
			render(w, r, models.PageData{View: "admin", AdminView: "product-edit", Title: "新建商品 - Shoper", AllCategories: cats, Message: "请填写必填字段（名称、品牌、单位、价格）", MessageType: "error"})
			return
		}
		res, err := db.DB.Exec(`INSERT INTO products (slug, group_name, name, brand, brand_logo, unit, price_cents, description, body) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			slug, "", name, brand, brandLogo, unit, int64(priceYuan*100), desc, body)
		if err != nil {
			cats, _ := db.GetAllCategories()
			render(w, r, models.PageData{View: "admin", AdminView: "product-edit", Title: "新建商品 - Shoper", AllCategories: cats, Message: "创建失败：" + err.Error(), MessageType: "error"})
			return
		}
		pid, _ := res.LastInsertId()
		if catID > 0 {
			db.DB.Exec(`UPDATE products SET group_name = (SELECT name FROM categories WHERE id = ?) WHERE id = ?`, catID, pid)
		}
		_ = db.ReplaceProductImages(pid, r.FormValue("images_payload"))
		var tagIDs []int64
		for _, idStr := range r.Form["tag_ids"] {
			if tid, err := strconv.ParseInt(idStr, 10, 64); err == nil && tid > 0 {
				tagIDs = append(tagIDs, tid)
			}
		}
		if jsonStr := strings.TrimSpace(r.FormValue("selected_tags_json")); jsonStr != "" && jsonStr != "null" {
			var ids []int64
			if err := json.Unmarshal([]byte(jsonStr), &ids); err == nil {
				tagIDs = ids
			}
		}
		db.ReplaceProductTags(pid, tagIDs)
		http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
		return
	}
	cats, _ := db.GetAllCategories()
	render(w, r, models.PageData{View: "admin", AdminView: "product-edit", Title: "新建商品 - Shoper", AllCategories: cats})
}

func adminProductEditHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/admin/product/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodPost {
		if r.FormValue("delete") == "1" {
			_, _ = db.DB.Exec(`DELETE FROM product_images WHERE product_id = ?`, id)
			_, _ = db.DB.Exec(`DELETE FROM products WHERE id = ?`, id)
			http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
			return
		}

		slug := strings.TrimSpace(r.FormValue("slug"))
		name := strings.TrimSpace(r.FormValue("name"))
		brand := strings.TrimSpace(r.FormValue("brand"))
		brandLogo := strings.TrimSpace(r.FormValue("brand_logo"))
		unit := strings.TrimSpace(r.FormValue("unit"))
		priceYuan, _ := strconv.ParseFloat(strings.TrimSpace(r.FormValue("price")), 64)
		desc := strings.TrimSpace(r.FormValue("description"))
		body := strings.TrimSpace(r.FormValue("body"))
		catID, _ := strconv.ParseInt(r.FormValue("category_id"), 10, 64)
		_, _ = db.DB.Exec(`UPDATE products SET slug=?, name=?, brand=?, brand_logo=?, unit=?, price_cents=?, description=?, body=? WHERE id=?`,
			slug, name, brand, brandLogo, unit, int64(priceYuan*100), desc, body, id)
		if catID > 0 {
			db.DB.Exec(`UPDATE products SET group_name = (SELECT name FROM categories WHERE id = ?) WHERE id = ?`, catID, id)
		}
		_ = db.ReplaceProductImages(id, r.FormValue("images_payload"))

		imgOrder := strings.TrimSpace(r.FormValue("img_order"))
		if imgOrder != "" {
			urls := strings.Split(imgOrder, ",")
			for i, url := range urls {
				url = strings.TrimSpace(url)
				if url == "" {
					continue
				}
				db.DB.Exec(`UPDATE product_images SET sort_order = ? WHERE product_id = ? AND image_url = ?`, i, id, url)
			}
		}

		var tagIDs []int64
		for _, idStr := range r.Form["tag_ids"] {
			if tid, err := strconv.ParseInt(idStr, 10, 64); err == nil && tid > 0 {
				tagIDs = append(tagIDs, tid)
			}
		}
		if jsonStr := strings.TrimSpace(r.FormValue("selected_tags_json")); jsonStr != "" && jsonStr != "null" {
			var ids []int64
			if err := json.Unmarshal([]byte(jsonStr), &ids); err == nil {
				tagIDs = ids
			}
		}
		db.ReplaceProductTags(id, tagIDs)

		http.Redirect(w, r, "/admin/products", http.StatusSeeOther)
		return
	}

	product, err := db.GetProductByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	cats, _ := db.GetAllCategories()
	render(w, r, models.PageData{View: "admin", AdminView: "product-edit", Title: "编辑商品 - Shoper", EditProduct: product, AllCategories: cats})
}

func adminTagsByCategoryHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	catID, _ := strconv.ParseInt(r.URL.Query().Get("category_id"), 10, 64)
	tags, _ := db.GetTagsByCategory(catID)
	w.Header().Set("Content-Type", "application/json")
	type tagJSON struct {
		ID   int64  `json:"id"`
		Name string `json:"name"`
	}
	var result []tagJSON
	for _, t := range tags {
		result = append(result, tagJSON{ID: t.ID, Name: t.Name})
	}
	if result == nil {
		result = []tagJSON{}
	}
	json.NewEncoder(w).Encode(result)
}
