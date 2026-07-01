package db

import "shoper/models"

func GetAllBrands() ([]models.Brand, error) {
	rows, err := DB.Query(`SELECT id, name, logo, description, sort_order FROM brands ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	brands := make([]models.Brand, 0)
	for rows.Next() {
		var b models.Brand
		if err := rows.Scan(&b.ID, &b.Name, &b.Logo, &b.Description, &b.SortOrder); err != nil {
			return nil, err
		}
		brands = append(brands, b)
	}
	return brands, rows.Err()
}

func BrandNameExists(name string, excludeID int64) bool {
	var count int
	_ = DB.QueryRow(`SELECT COUNT(*) FROM brands WHERE lower(name) = lower(?) AND id != ?`, name, excludeID).Scan(&count)
	return count > 0
}
