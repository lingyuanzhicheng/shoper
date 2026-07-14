package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
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
	if r.Method == http.MethodPost && !middleware.ValidateCSRF(r) {
		http.Error(w, "csrf token invalid", http.StatusForbidden)
		return
	}
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	tradeEnabled := db.GetSetting("feature_trade") != "disabled"
	var media []models.MediaFile

	// 子目录 -> 类型映射，扫描所有子目录文件
	subdirs := []string{"product", "brand", "voucher", "extra"}
	type fileEntry struct {
		filename string
		subdir   string
	}
	var allFiles []fileEntry
	for _, sub := range subdirs {
		entries, err := os.ReadDir(filepath.Join("uploads", sub))
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			allFiles = append(allFiles, fileEntry{filename: entry.Name(), subdir: sub})
		}
	}
	// 同时扫描根目录历史文件（未迁移的兼容场景）
	rootEntries, rootErr := os.ReadDir("uploads")
	if rootErr == nil {
		for _, entry := range rootEntries {
			if entry.IsDir() {
				continue
			}
			allFiles = append(allFiles, fileEntry{filename: entry.Name(), subdir: ""})
		}
	}

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
	for _, fe := range allFiles {
		filename := fe.filename
		// 优先按子目录推断类型，DB 引用作交叉验证
		mtype := "extra"
		tname := "额外配图"
		if fe.subdir == "product" || productImages[filename] {
			mtype = "product"
			tname = "商品配图"
		} else if fe.subdir == "brand" || brandLogos[filename] {
			mtype = "brand"
			tname = "品牌图标"
		} else if vtype, ok := voucherImages[filename]; ok {
			mtype = vtype
			tname = voucherTypeNames[vtype]
			if tname == "" {
				tname = "订单凭证"
			}
		} else if fe.subdir == "voucher" {
			mtype = "voucher"
			tname = "订单凭证"
		}
		// URL 根据子目录前缀生成；根目录文件保持原 URL（兼容旧数据）
		urlPath := "/uploads/" + filename
		if fe.subdir != "" {
			urlPath = "/uploads/" + fe.subdir + "/" + filename
		}
		// 交易功能关闭时隐藏下单凭证、送货凭证、退货凭证、结账记录
		if !tradeEnabled && (mtype == "voucher" || mtype == "delivery" || mtype == "return" || mtype == "statement") {
			continue
		}
		media = append(media, models.MediaFile{Filename: filename, URL: urlPath, Type: mtype, TypeName: tname})
	}
	if r.Method == http.MethodPost {
		fn := r.FormValue("delete")
		if fn != "" {
			// 删除时从所有可能位置尝试移除（子目录优先）
			base := filepath.Base(fn)
			for _, sub := range []string{"product", "brand", "voucher", "extra"} {
				os.Remove(filepath.Join("uploads", sub, base))
			}
			os.Remove(filepath.Join("uploads", base))
		}
		// 优先按表单 type 字段回跳到对应筛选视图，保留原分类
		redirectURL := "/admin/media"
		if t := strings.TrimSpace(r.FormValue("type")); t != "" {
			redirectURL = "/admin/media?type=" + t
		} else if ref := r.Referer(); ref != "" {
			// fallback：从 Referer 提取 type 参数
			if u, err := url.Parse(ref); err == nil {
				if t := u.Query().Get("type"); t != "" {
					redirectURL = "/admin/media?type=" + t
				}
			}
		}
		http.Redirect(w, r, redirectURL, http.StatusSeeOther)
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
	if !middleware.IsAdmin(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	if !middleware.ValidateCSRF(r) {
		http.Error(w, "csrf token invalid", http.StatusForbidden)
		return
	}
	kind := strings.TrimSpace(r.FormValue("type"))
	switch kind {
	case "product", "brand", "voucher", "extra":
		// ok
	default:
		kind = "extra"
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
	dir := filepath.Join("uploads", kind)
	if err := os.MkdirAll(dir, 0755); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer dst.Close()
	io.Copy(dst, file)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"url": "/uploads/" + kind + "/" + name})
}
