package db

import "shoper/models"

func ListOrdersPaged(page, size int) ([]models.Order, int, error) {
	if size <= 0 {
		size = 30
	}
	if page < 1 {
		page = 1
	}
	var total int
	if err := DB.QueryRow(`SELECT COUNT(*) FROM orders`).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []models.Order{}, 0, nil
	}
	offset := (page - 1) * size
	rows, err := DB.Query(`SELECT id, hash, contact_name, phone, address, community, building, unit_no, room, notes, status, archived, discount_cents, paid_cents, total_cents, created_at FROM orders ORDER BY id DESC LIMIT ? OFFSET ?`, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var orders []models.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		orders = append(orders, o)
	}
	return orders, total, rows.Err()
}
