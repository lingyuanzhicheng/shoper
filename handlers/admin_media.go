package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"shoper/db"
	"shoper/middleware"
	"shoper/models"
)

func adminMediaHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	tradeEnabled := db.GetSetting("feature_trade") != "disabled"
	var media []models.MediaFile

	entries, err := os.ReadDir("uploads")
	if err == nil {
		productImages := make(map[string]bool)
		rows, err := db.DB.Query(`SELECT image_url FROM product_images`)
		if err == nil {
			defer rows.Close()
			for rows.Next() {
				var url string
				rows.Scan(&url)
				if f := filepath.Base(url); f != "" && f != "." {
					productImages[f] = true
				}
			}
		}
		brandLogos := make(map[string]bool)
		brows, err := db.DB.Query(`SELECT logo FROM brands WHERE logo != ''`)
		if err == nil {
			defer brows.Close()
			for brows.Next() {
				var logo string
				brows.Scan(&logo)
				if f := filepath.Base(logo); f != "" && f != "." {
					brandLogos[f] = true
				}
			}
		}
		voucherImages := make(map[string]string)
		vrows, err := db.DB.Query(`SELECT image_url, type FROM order_vouchers WHERE image_url != ''`)
		if err == nil {
			defer vrows.Close()
			for vrows.Next() {
				var imgURL, vtype string
				vrows.Scan(&imgURL, &vtype)
				if f := filepath.Base(imgURL); f != "" && f != "." {
					voucherImages[f] = vtype
				}
			}
		}
		voucherTypeNames := map[string]string{
			"voucher":   "下单凭证",
			"delivery":  "送货凭证",
			"return":    "退货凭证",
			"statement": "结账记录",
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			filename := entry.Name()
			url := "/uploads/" + filename
			mtype := "extra"
			tname := "额外配图"
			if productImages[filename] {
				mtype = "product"
				tname = "商品配图"
			} else if brandLogos[filename] {
				mtype = "brand"
				tname = "品牌图标"
			} else if vtype, ok := voucherImages[filename]; ok {
			mtype = vtype
			tname = voucherTypeNames[vtype]
			if tname == "" {
				tname = "订单凭证"
			}
		}
		// 交易功能关闭时隐藏下单凭证、送货凭证、退货凭证、结账记录
		if !tradeEnabled && (mtype == "voucher" || mtype == "delivery" || mtype == "return" || mtype == "statement") {
			continue
		}
		media = append(media, models.MediaFile{Filename: filename, URL: url, Type: mtype, TypeName: tname})
		}
	}
	if r.Method == http.MethodPost {
		fn := r.FormValue("delete")
		if fn != "" {
			os.Remove(filepath.Join("uploads", filepath.Base(fn)))
		}
		http.Redirect(w, r, "/admin/media", http.StatusSeeOther)
		return
	}
	filterType := r.URL.Query().Get("type")
	var filtered []models.MediaFile
	for _, m := range media {
		if filterType == "" || m.Type == filterType {
			filtered = append(filtered, m)
		}
	}
	if filtered == nil {
		filtered = []models.MediaFile{}
	}
	page, size := parsePage(r)
	total := len(filtered)
	start := (page - 1) * size
	if start < 0 {
		start = 0
	}
	if start > total {
		start = total
	}
	end := start + size
	if end > total {
		end = total
	}
	paged := filtered[start:end]
	render(w, r, models.PageData{
		View:       "admin",
		AdminView:  "media",
		Title:      "媒体管理 - " + pn,
		Query:      filterType,
		MediaFiles: paged,
		Pagination: buildPagination("/admin/media", filterQuery(r), page, size, total),
	})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	r.ParseMultipartForm(10 << 20)
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	defer file.Close()
	ext := filepath.Ext(header.Filename)
	if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".gif" && ext != ".webp" {
		http.Error(w, "unsupported file type", 400)
		return
	}
	name := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dst, err := os.Create(filepath.Join("uploads", name))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer dst.Close()
	io.Copy(dst, file)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": "/uploads/" + name})
}
