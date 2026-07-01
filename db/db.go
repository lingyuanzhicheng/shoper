package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(database *sql.DB) error {
	DB = database
	return initDB()
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
			unit TEXT NOT NULL,
			price_cents INTEGER NOT NULL,
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
