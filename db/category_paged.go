package db

import "shoper/models"

func GetAllCategoriesPaged(page, size int) ([]models.Category, int, error) {
	if size <= 0 {
		size = 30
	}
	if page < 1 {
		page = 1
	}
	var total int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM categories`).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []models.Category{}, 0, nil
	}
	offset := (page - 1) * size
	rows, err := DB.Query(`SELECT id, name, COALESCE(parent_id, 0), sort_order FROM categories ORDER BY sort_order, id LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var all []models.Category
	for rows.Next() {
		var c models.Category
		var pid int64
		if err := rows.Scan(&c.ID, &c.Name, &pid, &c.SortOrder); err != nil {
			return nil, 0, err
		}
		if pid > 0 {
			c.ParentID = &pid
		}
		all = append(all, c)
	}
	return all, total, rows.Err()
}
