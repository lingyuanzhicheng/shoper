package db

import (
	"strings"

	"shoper/models"
)

func buildProductWhere(q, group, brand string) (string, []any) {
	args := []any{}
	where := []string{}
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
	clause := ""
	if len(where) > 0 {
		clause = ` WHERE ` + strings.Join(where, ` AND `)
	}
	return clause, args
}

func ListProductsPaged(q, group, brand string, page, size int) ([]models.Product, int, error) {
	if size <= 0 {
		size = 30
	}
	if page < 1 {
		page = 1
	}
	clause, args := buildProductWhere(q, group, brand)
	var total int
	countErr := DB.QueryRow(`SELECT COUNT(*) FROM products`+clause, args...).Scan(&total)
	if countErr != nil {
		return nil, 0, countErr
	}
	if total == 0 {
		return []models.Product{}, 0, nil
	}
	offset := (page - 1) * size
	query := `SELECT id, slug, group_name, name, brand, brand_logo, unit, price_cents, description, body FROM products` + clause + ` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, size, offset)
	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var products []models.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, err
		}
		p.Images, _ = ProductImages(p.ID)
		p.Tags, _ = GetProductTags(p.ID)
		products = append(products, p)
	}
	return products, total, rows.Err()
}
