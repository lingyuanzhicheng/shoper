package db

import "shoper/models"

func GetAllBrandsPaged(page, size int) ([]models.Brand, int, error) {
	if size <= 0 {
		size = 30
	}
	if page < 1 {
		page = 1
	}
	var total int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM brands`).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []models.Brand{}, 0, nil
	}
	offset := (page - 1) * size
	rows, err := DB.Query(`SELECT id, name, logo, description, sort_order FROM brands ORDER BY sort_order, id LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	brands := make([]models.Brand, 0)
	for rows.Next() {
		var b models.Brand
		if err := rows.Scan(&b.ID, &b.Name, &b.Logo, &b.Description, &b.SortOrder); err != nil {
			return nil, 0, err
		}
		brands = append(brands, b)
	}
	return brands, total, rows.Err()
}
