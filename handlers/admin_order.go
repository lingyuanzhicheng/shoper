package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"shoper/db"
	"shoper/middleware"
	"shoper/models"
	"shoper/utils"
)

func adminOrdersHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	page, size := parsePage(r)
	orders, total, err := db.ListOrdersPaged(page, size)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, models.PageData{View: "admin", AdminView: "orders", Title: "订单管理 - Shoper", Orders: orders, Pagination: buildPagination("/admin/orders", filterQuery(r), page, size, total)})
}

func adminOrderDetailHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	hash := strings.TrimPrefix(r.URL.Path, "/admin/orders/")
	if hash == "" {
		http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
		return
	}
	order, err := db.GetOrderByHash(hash)
	if err != nil {
		http.Error(w, "order not found", 404)
		return
	}
	products, _ := db.ListProducts("", "", "", 0)
	cats, _ := db.GetAllCategories()
	for i := range cats {
		cats[i].Tags, _ = db.GetTagsByCategory(cats[i].ID)
	}
	order.Vouchers, _ = db.GetOrderVouchers(hash)
	render(w, r, models.PageData{View: "admin", AdminView: "order-detail", Title: "订单详情 - " + hash, Order: order, AllProducts: products, AllCategories: cats})
}

func adminOrderUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
		return
	}
	if !middleware.ValidateCSRF(r) {
		http.Error(w, "csrf token invalid", http.StatusForbidden)
		return
	}
	hash := strings.TrimPrefix(r.URL.Path, "/admin/order/update/")
	status := strings.TrimSpace(r.FormValue("status"))
	if !models.IsValidOrderStatus(status) {
		http.Error(w, "invalid order status", http.StatusBadRequest)
		return
	}
	discountCents, _ := utils.ParseYuanToCents(r.FormValue("discount"))
	paidCents, _ := utils.ParseYuanToCents(r.FormValue("paid"))
	_ = r.FormValue("total") // total 由 itemsTotal 计算，不直接解析
	contactName := strings.TrimSpace(r.FormValue("contact_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	community := strings.TrimSpace(r.FormValue("community"))
	building := strings.TrimSpace(r.FormValue("building"))
	unitNo := strings.TrimSpace(r.FormValue("unit_no"))
	room := strings.TrimSpace(r.FormValue("room"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	archived := 0
	if r.FormValue("archived") == "1" {
		archived = 1
	}
	order, _ := db.GetOrderByHash(hash)
	deleteSet := map[string]bool{}
	for _, id := range r.Form["delete_item"] {
		deleteSet[id] = true
	}
	ids := r.Form["item_id"]
	qtys := r.Form["item_qty"]
	prices := r.Form["item_price"]
	units := r.Form["item_unit"]
	sorts := r.Form["item_sort"]
	deliveredSet := map[string]bool{}
	for _, id := range r.Form["delivered_item"] {
		deliveredSet[id] = true
	}
	returnSet := map[string]bool{}
	for _, id := range r.Form["return_item"] {
		returnSet[id] = true
	}
	productIDs := r.Form["item_product_id"]
	names := r.Form["item_name"]
	brands := r.Form["item_brand"]
	itemsTotal := int64(0)
	for i, idStr := range ids {
		itemID, _ := strconv.ParseInt(idStr, 10, 64)
		if deleteSet[idStr] {
			if itemID > 0 {
				_, _ = db.DB.Exec(`DELETE FROM order_items WHERE id = ? AND order_id = ?`, itemID, order.ID)
			}
			continue
		}
		qty, _ := strconv.Atoi(utils.ValueAt(qtys, i))
		priceCents, _ := utils.ParseYuanToCents(utils.ValueAt(prices, i))
		unit := strings.TrimSpace(utils.ValueAt(units, i))
		sortOrder, _ := strconv.Atoi(utils.ValueAt(sorts, i))
		isReturn := 0
		if returnSet[idStr] {
			isReturn = 1
		}
		delivered := 0
		if deliveredSet[idStr] {
			delivered = 1
		}
		if qty < 1 {
			qty = 1
		}
		if unit == "" {
			unit = "件"
		}
		lineCents := int64(qty) * priceCents
		if isReturn == 1 {
			lineCents = -lineCents
		}
		if itemID <= 0 {
			productID, _ := strconv.ParseInt(utils.ValueAt(productIDs, i), 10, 64)
			var pid any
			if productID > 0 {
				pid = productID
			}
			name := strings.TrimSpace(utils.ValueAt(names, i))
			if name == "" {
				name = "自定义商品"
			}
			brand := strings.TrimSpace(utils.ValueAt(brands, i))
			if brand == "" {
				brand = "商议"
			}
			_, _ = db.DB.Exec(`INSERT INTO order_items (order_id, product_id, product_name, brand, unit, qty, price_cents, line_cents, sort_order, delivered, is_return) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, order.ID, pid, name, brand, unit, qty, priceCents, lineCents, sortOrder, delivered, isReturn)
			itemsTotal += lineCents
			continue
		}
		itemsTotal += lineCents
		name := strings.TrimSpace(utils.ValueAt(names, i))
		if name == "" {
			name = "自定义商品"
		}
		brand := strings.TrimSpace(utils.ValueAt(brands, i))
		if brand == "" {
			brand = "商议"
		}
		_, _ = db.DB.Exec(`UPDATE order_items SET product_name = ?, brand = ?, unit = ?, qty = ?, price_cents = ?, line_cents = ?, sort_order = ?, delivered = ?, is_return = ? WHERE id = ? AND order_id = ?`, name, brand, unit, qty, priceCents, lineCents, sortOrder, delivered, isReturn, itemID, order.ID)
	}
	if itemsTotal == 0 {
		itemsTotal = 0 + discountCents
	}
	if discountCents < 0 {
		discountCents = 0
	}
	finalTotal := itemsTotal - discountCents
	if finalTotal < 0 {
		finalTotal = 0
	}
	address := community + " " + building + " " + unitNo + " " + room
	_, err := db.DB.Exec(`UPDATE orders SET status = ?, total_cents = ?, archived = ?, discount_cents = ?, paid_cents = ?, contact_name = ?, phone = ?, address = ?, community = ?, building = ?, unit_no = ?, room = ?, notes = ? WHERE hash = ?`, status, finalTotal, archived, discountCents, paidCents, contactName, phone, address, community, building, unitNo, room, notes, hash)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if r.URL.Query().Get("ajax") == "1" {
		dueCents := finalTotal - paidCents
		if dueCents < 0 {
			dueCents = 0
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"ok":             true,
			"total_cents":    finalTotal,
			"discount_cents": discountCents,
			"paid_cents":     paidCents,
			"original_total": itemsTotal,
			"due_cents":      dueCents,
		})
		return
	}
	http.Redirect(w, r, "/admin/orders/"+hash, http.StatusSeeOther)
}

func adminOrderCustomerUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
		return
	}
	if !middleware.ValidateCSRF(r) {
		http.Error(w, "csrf token invalid", http.StatusForbidden)
		return
	}
	hash := strings.TrimPrefix(r.URL.Path, "/admin/order/customer/update/")
	contactName := strings.TrimSpace(r.FormValue("contact_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	community := strings.TrimSpace(r.FormValue("community"))
	building := strings.TrimSpace(r.FormValue("building"))
	unitNo := strings.TrimSpace(r.FormValue("unit_no"))
	room := strings.TrimSpace(r.FormValue("room"))
	notes := strings.TrimSpace(r.FormValue("notes"))
	_, err := db.DB.Exec(`UPDATE orders SET contact_name=?, phone=?, community=?, building=?, unit_no=?, room=?, notes=? WHERE hash=?`,
		contactName, phone, community, building, unitNo, room, notes, hash)
	if r.URL.Query().Get("ajax") == "1" {
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.Write([]byte(`{"ok":false,"error":"保存失败"}`))
		} else {
			w.Write([]byte(`{"ok":true}`))
		}
		return
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/admin/orders/"+hash, http.StatusSeeOther)
}

func adminOrderStatusUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
		return
	}
	if !middleware.ValidateCSRF(r) {
		http.Error(w, "csrf token invalid", http.StatusForbidden)
		return
	}
	hash := strings.TrimPrefix(r.URL.Path, "/admin/order/status/update/")
	status := strings.TrimSpace(r.FormValue("status"))
	if !models.IsValidOrderStatus(status) {
		http.Error(w, "invalid order status", http.StatusBadRequest)
		return
	}
	_, err := db.DB.Exec(`UPDATE orders SET status = ? WHERE hash = ?`, status, hash)
	if r.URL.Query().Get("ajax") == "1" {
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.Write([]byte(`{"ok":false}`))
		} else {
			w.Write([]byte(`{"ok":true}`))
		}
		return
	}
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	http.Redirect(w, r, "/admin/orders/"+hash, http.StatusSeeOther)
}

func adminOrderArchiveHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	hash := strings.TrimPrefix(r.URL.Path, "/admin/order/archive/")
	_, _ = db.DB.Exec(`UPDATE orders SET archived = 1 WHERE hash = ?`, hash)
	http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
}

func adminOrderDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	hash := strings.TrimPrefix(r.URL.Path, "/admin/order/delete/")
	var id int64
	_ = db.DB.QueryRow(`SELECT id FROM orders WHERE hash = ?`, hash).Scan(&id)
	if id > 0 {
		_, _ = db.DB.Exec(`DELETE FROM order_items WHERE order_id = ?`, id)
		_, _ = db.DB.Exec(`DELETE FROM orders WHERE id = ?`, id)
	}
	http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
}

func adminOrderVoucherAddHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) || r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
		return
	}
	if !middleware.ValidateCSRF(r) {
		http.Error(w, "csrf token invalid", http.StatusForbidden)
		return
	}
	hash := r.FormValue("order_hash")
	vtype := r.FormValue("vtype")
	desc := r.FormValue("description")
	createdAt := r.FormValue("created_at")
	amountCents := int64(0)
	isRefund := int64(0)
	if vtype == "statement" {
		amountCents, _ = utils.ParseYuanToCents(r.FormValue("amount"))
		if r.FormValue("is_refund") == "1" {
			isRefund = 1
		}
	}
	imageURL := r.FormValue("image_url")
	if imageURL == "" {
		if r.URL.Query().Get("ajax") == "1" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok":false,"error":"请上传凭证图片"}`))
			return
		}
		http.Redirect(w, r, "/admin/orders/"+hash, http.StatusSeeOther)
		return
	}
	if strings.HasPrefix(imageURL, "data:") {
		if saved, err := utils.SaveImageDataURL(imageURL, "voucher"); err == nil {
			imageURL = saved
		}
	}
	res, err := db.DB.Exec(`INSERT INTO order_vouchers (order_hash, type, image_url, description, amount_cents, is_refund, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		hash, vtype, imageURL, desc, amountCents, isRefund, createdAt)
	if r.URL.Query().Get("ajax") == "1" {
		w.Header().Set("Content-Type", "application/json")
		if err != nil {
			w.Write([]byte(`{"ok":false,"error":"数据库错误"}`))
			return
		}
		id, _ := res.LastInsertId()
		fmt.Fprintf(w, `{"ok":true,"id":%d,"type":"%s","image_url":"%s","description":"%s","amount_cents":%d,"is_refund":%v,"created_at":"%s","order_hash":"%s"}`,
			id, vtype, imageURL, desc, amountCents, isRefund, createdAt, hash)
		return
	}
	http.Redirect(w, r, "/admin/orders/"+hash+"?tab="+vtype, http.StatusSeeOther)
}

func adminOrderVoucherDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) || r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
		return
	}
	if !middleware.ValidateCSRF(r) {
		http.Error(w, "csrf token invalid", http.StatusForbidden)
		return
	}
	idStr := r.FormValue("id")
	hash := r.FormValue("order_hash")
	vtype := r.FormValue("vtype")
	if vtype == "" {
		vtype = "items"
	}
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if id > 0 {
		db.DB.Exec(`DELETE FROM order_vouchers WHERE id = ?`, id)
	}
	http.Redirect(w, r, "/admin/orders/"+hash+"?tab="+vtype, http.StatusSeeOther)
}
