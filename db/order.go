package db

import "shoper/models"

type orderScanner interface {
	Scan(dest ...any) error
}

func scanOrder(s orderScanner) (models.Order, error) {
	var o models.Order
	var archived int
	err := s.Scan(&o.ID, &o.Hash, &o.ContactName, &o.Phone, &o.Address, &o.Community, &o.Building, &o.UnitNo, &o.Room, &o.Notes, &o.Status, &archived, &o.DiscountCents, &o.PaidCents, &o.TotalCents, &o.CreatedAt)
	o.Archived = archived == 1
	return o, err
}

func GetOrderForCustomer(hash, phone string) (models.Order, error) {
	order, err := getOrderByQuery(`SELECT id, hash, contact_name, phone, address, community, building, unit_no, room, notes, status, archived, discount_cents, paid_cents, total_cents, created_at FROM orders WHERE hash = ? AND phone = ? AND archived = 0`, hash, phone)
	if err != nil {
		return models.Order{}, err
	}
	order.Items, _ = OrderItems(order.ID)
	return order, nil
}

// GetOrderByHashForCustomer looks up an order by hash only (without phone or
// archived filtering), returning all columns including paid_cents. The caller
// is responsible for checking phone match and status.
func GetOrderByHashForCustomer(hash string) (models.Order, error) {
	order, err := getOrderByQuery(`SELECT id, hash, contact_name, phone, address, community, building, unit_no, room, notes, status, archived, discount_cents, paid_cents, total_cents, created_at FROM orders WHERE hash = ?`, hash)
	if err != nil {
		return models.Order{}, err
	}
	order.Items, _ = OrderItems(order.ID)
	return order, nil
}

func GetOrderByHash(hash string) (models.Order, error) {
	order, err := getOrderByQuery(`SELECT id, hash, contact_name, phone, address, community, building, unit_no, room, notes, status, archived, discount_cents, paid_cents, total_cents, created_at FROM orders WHERE hash = ?`, hash)
	if err != nil {
		return models.Order{}, err
	}
	order.Items, _ = OrderItems(order.ID)
	return order, nil
}

func ListOrders() ([]models.Order, error) {
	rows, err := DB.Query(`SELECT id, hash, contact_name, phone, address, community, building, unit_no, room, notes, status, archived, discount_cents, paid_cents, total_cents, created_at FROM orders ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orders []models.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func getOrderByQuery(query string, args ...any) (models.Order, error) {
	row := DB.QueryRow(query, args...)
	return scanOrder(row)
}

func OrderItems(orderID int64) ([]models.OrderItem, error) {
	rows, err := DB.Query(`SELECT id, COALESCE(product_id, 0), product_name, brand, unit, qty, price_cents, line_cents, delivered, is_return FROM order_items WHERE order_id = ? ORDER BY sort_order, id`, orderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []models.OrderItem
	for rows.Next() {
		var item models.OrderItem
		var delivered, isReturn int
		if err := rows.Scan(&item.ID, &item.ProductID, &item.ProductName, &item.Brand, &item.Unit, &item.Qty, &item.PriceCents, &item.LineCents, &delivered, &isReturn); err != nil {
			return nil, err
		}
		item.Delivered = delivered == 1
		item.IsReturn = isReturn == 1
		if item.ProductID > 0 {
			if images, err := ProductImages(item.ProductID); err == nil && len(images) > 0 {
				item.Image = images[0]
			}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func GetOrderVouchers(hash string) ([]models.Voucher, error) {
	rows, err := DB.Query(`SELECT id, order_hash, type, image_url, description, amount_cents, is_refund, created_at FROM order_vouchers WHERE order_hash = ? ORDER BY created_at DESC`, hash)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var vouchers []models.Voucher
	for rows.Next() {
		var v models.Voucher
		var isRefund int64
		if err := rows.Scan(&v.ID, &v.OrderHash, &v.Type, &v.ImageURL, &v.Description, &v.AmountCents, &isRefund, &v.CreatedAt); err != nil {
			continue
		}
		v.IsRefund = isRefund != 0
		vouchers = append(vouchers, v)
	}
	if vouchers == nil {
		vouchers = []models.Voucher{}
	}
	return vouchers, nil
}
