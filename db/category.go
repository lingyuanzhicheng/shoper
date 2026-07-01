package db

import (
	"shoper/models"
)

func GetAllCategories() ([]models.Category, error) {
	rows, err := DB.Query(`SELECT id, name, COALESCE(parent_id, 0), sort_order FROM categories ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var all []models.Category
	for rows.Next() {
		var c models.Category
		var pid int64
		if err := rows.Scan(&c.ID, &c.Name, &pid, &c.SortOrder); err != nil {
			return nil, err
		}
		if pid > 0 {
			c.ParentID = &pid
		}
		all = append(all, c)
	}
	return all, nil
}

func GetCategoryByID(id int64) (*models.Category, error) {
	row := DB.QueryRow(`SELECT id, name, COALESCE(parent_id, 0), sort_order FROM categories WHERE id = ?`, id)
	var c models.Category
	var pid int64
	if err := row.Scan(&c.ID, &c.Name, &pid, &c.SortOrder); err != nil {
		return nil, err
	}
	if pid > 0 {
		c.ParentID = &pid
	}
	return &c, nil
}

func GetTagsByCategory(categoryID int64) ([]models.Tag, error) {
	rows, err := DB.Query(`SELECT id, name, category_id, sort_order FROM tags WHERE category_id = ? ORDER BY sort_order, id`, categoryID)
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

func GetCategoryByName(name string) (*models.Category, error) {
	if name == "" {
		return nil, nil
	}
	row := DB.QueryRow(`SELECT id, name, COALESCE(parent_id, 0), sort_order FROM categories WHERE name = ?`, name)
	var c models.Category
	var pid int64
	if err := row.Scan(&c.ID, &c.Name, &pid, &c.SortOrder); err != nil {
		return nil, nil
	}
	if pid > 0 {
		c.ParentID = &pid
	}
	return &c, nil
}

func LoadAllCategoriesWithTags() []models.Category {
	cats, _ := GetAllCategories()
	for i := range cats {
		cats[i].Tags, _ = GetTagsByCategory(cats[i].ID)
	}
	return cats
}

func CategoryNameExists(name string) bool {
	var count int
	_ = DB.QueryRow(`SELECT COUNT(*) FROM categories WHERE lower(name) = lower(?)`, name).Scan(&count)
	return count > 0
}

func TagNameExistsInCategory(categoryID int64, name string) bool {
	var count int
	_ = DB.QueryRow(`SELECT COUNT(*) FROM tags WHERE category_id = ? AND lower(name) = lower(?)`, categoryID, name).Scan(&count)
	return count > 0
}
