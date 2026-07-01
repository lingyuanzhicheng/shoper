package handlers

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"shoper/db"
	"shoper/middleware"
	"shoper/models"
)

func readCart(r *http.Request) map[int64]int {
	cart := map[int64]int{}
	cookie, err := r.Cookie("cart")
	if err != nil || cookie.Value == "" {
		return cart
	}
	decoded, err := base64.RawURLEncoding.DecodeString(cookie.Value)
	if err != nil {
		return cart
	}
	var raw map[string]int
	if err := json.Unmarshal(decoded, &raw); err != nil {
		return cart
	}
	for k, v := range raw {
		id, _ := strconv.ParseInt(k, 10, 64)
		if id > 0 && v > 0 {
			cart[id] = v
		}
	}
	return cart
}

func writeCart(w http.ResponseWriter, cart map[int64]int) {
	raw := map[string]int{}
	for id, qty := range cart {
		if qty > 0 {
			raw[strconv.FormatInt(id, 10)] = qty
		}
	}
	b, _ := json.Marshal(raw)
	http.SetCookie(w, &http.Cookie{Name: "cart", Value: base64.RawURLEncoding.EncodeToString(b), Path: "/", MaxAge: 86400 * 30, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

func cartLines(r *http.Request) ([]models.CartLine, int64, error) {
	cart := readCart(r)
	var lines []models.CartLine
	var total int64
	for id, qty := range cart {
		p, err := db.GetProductByID(id)
		if err != nil {
			continue
		}
		sub := int64(qty) * p.PriceCents
		total += sub
		lines = append(lines, models.CartLine{Product: p, Qty: qty, SubCents: sub})
	}
	return lines, total, nil
}

func cartCount(r *http.Request) int {
	return len(readCart(r))
}

func cartHandler(w http.ResponseWriter, r *http.Request) {
	lines, total, err := cartLines(r)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	render(w, r, models.PageData{View: "cart", Title: "购物车 - Shoper", CartLines: lines, CartTotal: total})
}

func addCartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}
	id, _ := strconv.ParseInt(r.FormValue("product_id"), 10, 64)
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	if qty < 1 {
		qty = 1
	}
	cart := readCart(r)
	if r.FormValue("mode") == "set" {
		cart[id] = qty
	} else {
		cart[id] += qty
	}
	writeCart(w, cart)
	if r.FormValue("next") != "" {
		http.Redirect(w, r, r.FormValue("next"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func removeCartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}
	id, _ := strconv.ParseInt(r.FormValue("product_id"), 10, 64)
	cart := readCart(r)
	delete(cart, id)
	writeCart(w, cart)
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func updateCartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}
	id, _ := strconv.ParseInt(r.FormValue("product_id"), 10, 64)
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	cart := readCart(r)
	if qty <= 0 {
		delete(cart, id)
	} else {
		cart[id] = qty
	}
	writeCart(w, cart)
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func createOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}
	phone := strings.TrimSpace(r.FormValue("phone"))
	contactName := strings.TrimSpace(r.FormValue("contact_name"))
	community := strings.TrimSpace(r.FormValue("community"))
	building := strings.TrimSpace(r.FormValue("building"))
	unitNo := strings.TrimSpace(r.FormValue("unit_no"))
	room := strings.TrimSpace(r.FormValue("room"))
	notes := strings.TrimSpace(r.FormValue("notes"))

	if contactName == "" || phone == "" || community == "" || building == "" || unitNo == "" || room == "" {
		lines, total, _ := cartLines(r)
		render(w, r, models.PageData{View: "cart", Title: "购物车 - Shoper", CartLines: lines, CartTotal: total, Message: "请填写完整的联系方式和地址信息", MessageType: "error", Community: community, Building: building, UnitNo: unitNo, Room: room, Notes: notes})
		return
	}

	hasFullAddr := community != "" && building != "" && unitNo != "" && room != ""
	if !hasFullAddr && notes == "" {
		lines, total, _ := cartLines(r)
		render(w, r, models.PageData{View: "cart", Title: "购物车 - Shoper", CartLines: lines, CartTotal: total, Message: "地址信息不完整时请填写备注说明", MessageType: "warning", Community: community, Building: building, UnitNo: unitNo, Room: room, Notes: notes})
		return
	}

	lines, total, err := cartLines(r)
	if err != nil || len(lines) == 0 {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}

	address := community + " " + building + " " + unitNo + " " + room
	hash := middleware.RandomHash()
	res, err := db.DB.Exec(`INSERT INTO orders (hash, contact_name, phone, address, community, building, unit_no, room, notes, status, discount_cents, total_cents, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		hash, contactName, phone, address, community, building, unitNo, room, notes, "待确认", 0, total, time.Now().Format("2006-01-02 15:04"))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	orderID, _ := res.LastInsertId()
	for _, line := range lines {
		_, _ = db.DB.Exec(`INSERT INTO order_items (order_id, product_id, product_name, brand, unit, qty, price_cents, line_cents) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			orderID, line.Product.ID, line.Product.Name, line.Product.Brand, line.Product.Unit, line.Qty, line.Product.PriceCents, line.SubCents)
	}
	writeCart(w, map[int64]int{})
	http.Redirect(w, r, "/track?hash="+hash+"&phone="+phone, http.StatusSeeOther)
}
