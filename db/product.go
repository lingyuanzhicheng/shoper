package db

import (
	"encoding/json"
	"shoper/models"
	"shoper/utils"
	"sort"
	"strings"
)

type productScanner interface {
	Scan(dest ...any) error
}

func scanProduct(s productScanner) (models.Product, error) {
	var p models.Product
	err := s.Scan(&p.ID, &p.Slug, &p.Group, &p.Name, &p.Brand, &p.BrandLogo, &p.Description, &p.Body)
	return p, err
}

// populateProductFields 从最低价格的型号填充 PriceCents 和 Unit（列表页显示起步价）
func populateProductFields(p *models.Product) {
	if len(p.Models) == 0 {
		return
	}
	min := p.Models[0]
	for _, m := range p.Models[1:] {
		if m.PriceCents < min.PriceCents {
			min = m
		}
	}
	p.PriceCents = min.PriceCents
	p.Unit = min.Unit
}

func ListProducts(q, group, brand string, limit int) ([]models.Product, error) {
	args := []any{}
	where := []string{}
	query := `SELECT id, slug, group_name, name, brand, brand_logo, description, body FROM products`
	if q != "" {
		where = append(where, `(name LIKE ? OR brand LIKE ? OR description LIKE ? OR group_name LIKE ?)`)
		like := "%" + q + "%"
		args = append(args, like, like, like, like)
	}
	if group != "" {
		where = append(where, `group_name = ?`)
		args = append(args, group)
	}
	if brand != "" {
		where = append(where, `brand = ?`)
		args = append(args, brand)
	}
	if len(where) > 0 {
		query += ` WHERE ` + strings.Join(where, ` AND `)
	}
	query += ` ORDER BY id DESC`
	if limit > 0 {
		query += ` LIMIT ?`
		args = append(args, limit)
	}
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []models.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		p.Images, _ = ProductImages(p.ID)
		p.Tags, _ = GetProductTags(p.ID)
		p.Models, _ = GetProductModels(p.ID)
		populateProductFields(&p)
		products = append(products, p)
	}
	return products, rows.Err()
}

func GetProductBySlug(slug string) (models.Product, error) {
	row := DB.QueryRow(`SELECT id, slug, group_name, name, brand, brand_logo, description, body FROM products WHERE slug = ?`, slug)
	p, err := scanProduct(row)
	if err != nil {
		return models.Product{}, err
	}
	p.Images, _ = ProductImages(p.ID)
	p.Tags, _ = GetProductTags(p.ID)
	p.Models, _ = GetProductModels(p.ID)
	populateProductFields(&p)
	return p, nil
}

func GetProductByID(id int64) (models.Product, error) {
	row := DB.QueryRow(`SELECT id, slug, group_name, name, brand, brand_logo, description, body FROM products WHERE id = ?`, id)
	p, err := scanProduct(row)
	if err != nil {
		return models.Product{}, err
	}
	p.Images, _ = ProductImages(p.ID)
	p.Tags, _ = GetProductTags(p.ID)
	p.Models, _ = GetProductModels(p.ID)
	populateProductFields(&p)
	return p, nil
}

func ProductGroups() ([]string, error) {
	rows, err := DB.Query(`SELECT DISTINCT group_name FROM products WHERE group_name != '' ORDER BY group_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []string
	for rows.Next() {
		var g string
		if err := rows.Scan(&g); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func ProductImages(productID int64) ([]string, error) {
	rows, err := DB.Query(`SELECT image_url FROM product_images WHERE product_id = ? ORDER BY sort_order, id`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var images []string
	for rows.Next() {
		var img string
		if err := rows.Scan(&img); err != nil {
			return nil, err
		}
		images = append(images, img)
	}
	return images, rows.Err()
}

func ReplaceProductImages(productID int64, payload string) error {
	if strings.TrimSpace(payload) == "" {
		return nil
	}
	var images []string
	if err := json.Unmarshal([]byte(payload), &images); err != nil {
		return err
	}
	_, _ = DB.Exec(`DELETE FROM product_images WHERE product_id = ?`, productID)
	for i, img := range images {
		img = strings.TrimSpace(img)
		if img == "" {
			continue
		}
		if strings.HasPrefix(img, "data:") {
			saved, err := utils.SaveImageDataURL(img)
			if err != nil {
				continue
			}
			img = saved
		}
		_, _ = DB.Exec(`INSERT INTO product_images (product_id, image_url, sort_order) VALUES (?, ?, ?)`, productID, img, i)
	}
	return nil
}

func GetProductTags(productID int64) ([]models.Tag, error) {
	rows, err := DB.Query(`SELECT t.id, t.name, t.category_id, t.sort_order FROM tags t JOIN product_tags pt ON t.id = pt.tag_id WHERE pt.product_id = ? ORDER BY t.sort_order, t.id`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.CategoryID, &t.SortOrder); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func ReplaceProductTags(productID int64, tagIDs []int64) {
	DB.Exec(`DELETE FROM product_tags WHERE product_id = ?`, productID)
	for _, tid := range tagIDs {
		DB.Exec(`INSERT OR IGNORE INTO product_tags (product_id, tag_id) VALUES (?, ?)`, productID, tid)
	}
}

func GetProductModels(productID int64) ([]models.ProductModel, error) {
	rows, err := DB.Query(`SELECT id, product_id, model_name, price_cents, unit, sort_order FROM product_models WHERE product_id = ? ORDER BY sort_order, id`, productID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []models.ProductModel
	for rows.Next() {
		var m models.ProductModel
		if err := rows.Scan(&m.ID, &m.ProductID, &m.ModelName, &m.PriceCents, &m.Unit, &m.SortOrder); err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func ReplaceProductModels(productID int64, ms []models.ProductModel) {
	DB.Exec(`DELETE FROM product_models WHERE product_id = ?`, productID)
	for i, m := range ms {
		DB.Exec(`INSERT INTO product_models (product_id, model_name, price_cents, unit, sort_order) VALUES (?, ?, ?, ?, ?)`, productID, m.ModelName, m.PriceCents, m.Unit, i)
	}
}

func ListProductsByIDs(ids []int64) ([]models.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	query := `SELECT id, slug, group_name, name, brand, brand_logo, description, body FROM products WHERE id IN (` + strings.Join(placeholders, ",") + `) ORDER BY id DESC`
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var products []models.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, err
		}
		p.Images, _ = ProductImages(p.ID)
		p.Tags, _ = GetProductTags(p.ID)
		p.Models, _ = GetProductModels(p.ID)
		populateProductFields(&p)
		products = append(products, p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	orderMap := make(map[int64]int, len(ids))
	for i, id := range ids {
		orderMap[id] = i
	}
	sort.Slice(products, func(i, j int) bool {
		return orderMap[products[i].ID] < orderMap[products[j].ID]
	})
	return products, nil
}
