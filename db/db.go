package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(database *sql.DB) error {
	DB = database
	if err := initDB(); err != nil {
		return err
	}
	migrateProductModels()
	migrateUploads()
	// 迁移 admin_phone/admin_key → admin_username/admin_password（幂等）
	if v := GetSetting("admin_phone"); v != "" {
		if GetSetting("admin_username") == "" {
			SetSetting("admin_username", v)
		}
		_, _ = DB.Exec(`DELETE FROM settings WHERE key = ?`, "admin_phone")
	}
	if v := GetSetting("admin_key"); v != "" {
		if GetSetting("admin_password") == "" {
			SetSetting("admin_password", v)
		}
		_, _ = DB.Exec(`DELETE FROM settings WHERE key = ?`, "admin_key")
	}
	return nil
}

func initDB() error {
	schema := []string{
		`CREATE TABLE IF NOT EXISTS products (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			group_name TEXT NOT NULL DEFAULT '综合商品',
			name TEXT NOT NULL,
			brand TEXT NOT NULL,
			brand_logo TEXT NOT NULL,
			description TEXT NOT NULL,
			body TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS product_images (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			image_url TEXT NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			parent_id INTEGER REFERENCES categories(id) ON DELETE SET NULL,
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			category_id INTEGER NOT NULL REFERENCES categories(id) ON DELETE CASCADE,
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS brands (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL UNIQUE,
			logo TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS orders (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hash TEXT NOT NULL UNIQUE,
			contact_name TEXT NOT NULL DEFAULT '',
			phone TEXT NOT NULL,
			address TEXT NOT NULL DEFAULT '',
			community TEXT NOT NULL DEFAULT '',
			building TEXT NOT NULL DEFAULT '',
			unit_no TEXT NOT NULL DEFAULT '',
			room TEXT NOT NULL DEFAULT '',
			notes TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT '待确认',
			archived INTEGER NOT NULL DEFAULT 0,
			discount_cents INTEGER NOT NULL DEFAULT 0,
			total_cents INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS order_items (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_id INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
			product_id INTEGER,
			product_name TEXT NOT NULL,
			brand TEXT NOT NULL,
			unit TEXT NOT NULL,
			qty INTEGER NOT NULL,
			price_cents INTEGER NOT NULL,
			line_cents INTEGER NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0,
			delivered INTEGER NOT NULL DEFAULT 0,
			is_return INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE TABLE IF NOT EXISTS order_vouchers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			order_hash TEXT NOT NULL,
			type TEXT NOT NULL,
			image_url TEXT NOT NULL DEFAULT '',
			description TEXT NOT NULL DEFAULT '',
			amount_cents INTEGER NOT NULL DEFAULT 0,
			is_refund INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS product_tags (
			product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			tag_id INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
			PRIMARY KEY (product_id, tag_id)
		)`,
		`CREATE TABLE IF NOT EXISTS product_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			product_id INTEGER NOT NULL REFERENCES products(id) ON DELETE CASCADE,
			model_name TEXT NOT NULL DEFAULT '',
			price_cents INTEGER NOT NULL DEFAULT 0,
			unit TEXT NOT NULL DEFAULT '',
			sort_order INTEGER NOT NULL DEFAULT 0
		)`,
	}
	_, _ = DB.Exec(`ALTER TABLE orders ADD COLUMN contact_name TEXT NOT NULL DEFAULT ''`)
	_, _ = DB.Exec(`ALTER TABLE orders ADD COLUMN discount_cents INTEGER NOT NULL DEFAULT 0`)
	_, _ = DB.Exec(`ALTER TABLE order_items ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0`)
	_, _ = DB.Exec(`ALTER TABLE order_items ADD COLUMN delivered INTEGER NOT NULL DEFAULT 0`)
	_, _ = DB.Exec(`ALTER TABLE order_items ADD COLUMN is_return INTEGER NOT NULL DEFAULT 0`)
	for _, q := range schema {
		if _, err := DB.Exec(q); err != nil {
			return err
		}
	}
	_, _ = DB.Exec(`ALTER TABLE products ADD COLUMN body TEXT NOT NULL DEFAULT ''`)
	_, _ = DB.Exec(`ALTER TABLE orders ADD COLUMN community TEXT NOT NULL DEFAULT ''`)
	_, _ = DB.Exec(`ALTER TABLE orders ADD COLUMN building TEXT NOT NULL DEFAULT ''`)
	_, _ = DB.Exec(`ALTER TABLE orders ADD COLUMN unit_no TEXT NOT NULL DEFAULT ''`)
	_, _ = DB.Exec(`ALTER TABLE orders ADD COLUMN room TEXT NOT NULL DEFAULT ''`)
	_, _ = DB.Exec(`ALTER TABLE orders ADD COLUMN notes TEXT NOT NULL DEFAULT ''`)
	_, _ = DB.Exec(`ALTER TABLE orders ADD COLUMN paid_cents INTEGER NOT NULL DEFAULT 0`)
	_, _ = DB.Exec(`ALTER TABLE products ADD COLUMN group_name TEXT NOT NULL DEFAULT '综合商品'`)
	return nil
}

func columnExists(table, column string) bool {
	var name string
	err := DB.QueryRow("SELECT name FROM pragma_table_info(?) WHERE name = ?", table, column).Scan(&name)
	return err == nil
}

// migrateProductModels 迁移旧的 price_cents/unit 到 product_models 表，
// 从商品名的 () 中提取型号名称，清理商品名，然后删除旧字段。
func migrateProductModels() {
	if !columnExists("products", "price_cents") {
		return
	}

	rows, err := DB.Query("SELECT id, name, price_cents, unit FROM products")
	if err != nil {
		return
	}
	type oldProd struct {
		ID         int64
		Name       string
		PriceCents int64
		Unit       string
	}
	var products []oldProd
	for rows.Next() {
		var p oldProd
		if err := rows.Scan(&p.ID, &p.Name, &p.PriceCents, &p.Unit); err != nil {
			continue
		}
		products = append(products, p)
	}
	rows.Close()

	for _, p := range products {
		modelName := ""
		cleanName := p.Name
		if idx := strings.Index(p.Name, "("); idx >= 0 {
			if endIdx := strings.Index(p.Name[idx:], ")"); endIdx > 0 {
				modelName = strings.TrimSpace(p.Name[idx+1 : idx+endIdx])
				cleanName = strings.TrimSpace(p.Name[:idx]) + strings.TrimSpace(p.Name[idx+endIdx+1:])
				cleanName = strings.TrimSpace(cleanName)
			}
		}
		if modelName == "" {
			modelName = "默认"
		}

		var count int
		DB.QueryRow("SELECT COUNT(*) FROM product_models WHERE product_id = ?", p.ID).Scan(&count)
		if count == 0 {
			DB.Exec("INSERT INTO product_models (product_id, model_name, price_cents, unit, sort_order) VALUES (?, ?, ?, ?, 0)",
				p.ID, modelName, p.PriceCents, p.Unit)
		}
		if cleanName != p.Name {
			DB.Exec("UPDATE products SET name = ? WHERE id = ?", cleanName, p.ID)
		}
	}

	_, _ = DB.Exec("ALTER TABLE products DROP COLUMN price_cents")
	_, _ = DB.Exec("ALTER TABLE products DROP COLUMN unit")
}

// migrateUploads 将 uploads/ 根目录下的历史文件按 DB 引用迁移到对应子目录。
// 已位于子目录的文件跳过；目标已存在时跳过；迁移幂等可重复执行。
func migrateUploads() {
	// 确保子目录存在
	for _, sub := range []string{"product", "brand", "voucher", "extra"} {
		_ = os.MkdirAll(filepath.Join("uploads", sub), 0755)
	}

	// 收集 DB 引用：basename -> 目标子目录
	type ref struct {
		basename string
		subdir   string
		urlCol   string // 用于更新 DB 的完整 URL 前缀
	}
	refs := []ref{}

	// product_images
	rows, err := DB.Query(`SELECT image_url FROM product_images`)
	if err == nil {
		for rows.Next() {
			var u string
			rows.Scan(&u)
			base := filepath.Base(u)
			if base != "" && base != "." && !strings.Contains(u, "/product/") {
				refs = append(refs, ref{basename: base, subdir: "product", urlCol: "/uploads/product/" + base})
			}
		}
		rows.Close()
	}

	// brands
	brows, err := DB.Query(`SELECT logo FROM brands WHERE logo != ''`)
	if err == nil {
		for brows.Next() {
			var u string
			brows.Scan(&u)
			base := filepath.Base(u)
			if base != "" && base != "." && !strings.Contains(u, "/brand/") {
				refs = append(refs, ref{basename: base, subdir: "brand", urlCol: "/uploads/brand/" + base})
			}
		}
		brows.Close()
	}

	// order_vouchers
	vrows, err := DB.Query(`SELECT image_url FROM order_vouchers WHERE image_url != ''`)
	if err == nil {
		for vrows.Next() {
			var u string
			vrows.Scan(&u)
			base := filepath.Base(u)
			if base != "" && base != "." && !strings.Contains(u, "/voucher/") {
				refs = append(refs, ref{basename: base, subdir: "voucher", urlCol: "/uploads/voucher/" + base})
			}
		}
		vrows.Close()
	}

	// 扫描 uploads/ 根目录文件
	entries, err := os.ReadDir("uploads")
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		filename := entry.Name()
		// 查找该文件对应的 DB 引用
		matched := false
		for _, r := range refs {
			if r.basename == filename {
				src := filepath.Join("uploads", filename)
				dst := filepath.Join("uploads", r.subdir, filename)
				// 若目标已存在则跳过文件移动，但仍尝试更新 DB
				if _, err := os.Stat(dst); os.IsNotExist(err) {
					if err := os.Rename(src, dst); err != nil {
						continue
					}
				}
				matched = true
				break
			}
		}
		if !matched {
			// 无 DB 引用的归到 extra
			src := filepath.Join("uploads", filename)
			dst := filepath.Join("uploads", "extra", filename)
			if _, err := os.Stat(dst); os.IsNotExist(err) {
				os.Rename(src, dst)
			}
		}
	}

	// 更新 DB 中的 image_url 字段
	for _, r := range refs {
		switch r.subdir {
		case "product":
			DB.Exec(`UPDATE product_images SET image_url = ? WHERE image_url = ?`, r.urlCol, "/uploads/"+r.basename)
		case "brand":
			DB.Exec(`UPDATE brands SET logo = ? WHERE logo = ?`, r.urlCol, "/uploads/"+r.basename)
		case "voucher":
			DB.Exec(`UPDATE order_vouchers SET image_url = ? WHERE image_url = ?`, r.urlCol, "/uploads/"+r.basename)
		}
	}
}
