package db

import (
	"database/sql"
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
