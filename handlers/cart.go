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

// readCart 读取购物车 cookie，返回 map[string]int，key 格式为 "productID:modelID"
func readCart(r *http.Request) map[string]int {
	cart := map[string]int{}
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
		if v > 0 {
			// 兼容旧格式（纯 productID）和新格式（productID:modelID）
			if !strings.Contains(k, ":") {
				k = k + ":0"
			}
			cart[k] = v
		}
	}
	return cart
}

func writeCart(w http.ResponseWriter, cart map[string]int) {
	raw := map[string]int{}
	for key, qty := range cart {
		if qty > 0 {
			raw[key] = qty
		}
	}
	b, _ := json.Marshal(raw)
	http.SetCookie(w, &http.Cookie{Name: "cart", Value: base64.RawURLEncoding.EncodeToString(b), Path: "/", MaxAge: 86400 * 30, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

// cartKey 生成购物车 key
func cartKey(productID, modelID int64) string {
	return strconv.FormatInt(productID, 10) + ":" + strconv.FormatInt(modelID, 10)
}

// parseCartKey 解析购物车 key，返回 productID 和 modelID
func parseCartKey(key string) (int64, int64) {
	parts := strings.SplitN(key, ":", 2)
	productID, _ := strconv.ParseInt(parts[0], 10, 64)
	modelID := int64(0)
	if len(parts) > 1 {
		modelID, _ = strconv.ParseInt(parts[1], 10, 64)
	}
	return productID, modelID
}

func cartLines(r *http.Request) ([]models.CartLine, int64, error) {
	cart := readCart(r)
	var lines []models.CartLine
	var total int64
	for key, qty := range cart {
		productID, modelID := parseCartKey(key)
		p, err := db.GetProductByID(productID)
		if err != nil {
			continue
		}
		// 找到选中的型号
		var model *models.ProductModel
		for i := range p.Models {
			if p.Models[i].ID == modelID {
				model = &p.Models[i]
				break
			}
		}
		if model == nil && len(p.Models) > 0 {
			model = &p.Models[0]
		}
		priceCents := int64(0)
		unit := ""
		modelName := ""
		if model != nil {
			priceCents = model.PriceCents
			unit = model.Unit
			modelName = model.ModelName
		}
		p.PriceCents = priceCents
		p.Unit = unit
		sub := int64(qty) * priceCents
		total += sub
		lines = append(lines, models.CartLine{Product: p, Qty: qty, SubCents: sub, ModelID: modelID, ModelName: modelName})
	}
	return lines, total, nil
}

func cartCount(r *http.Request) int {
	cart := readCart(r)
	count := 0
	for key := range cart {
		productID, _ := parseCartKey(key)
		if _, err := db.GetProductByID(productID); err == nil {
			count++
		}
	}
	return count
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
	modelID, _ := strconv.ParseInt(r.FormValue("model_id"), 10, 64)
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	if qty < 1 {
		qty = 1
	}
	key := cartKey(id, modelID)
	cart := readCart(r)
	if r.FormValue("mode") == "set" {
		cart[key] = qty
	} else {
		cart[key] += qty
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
	modelID, _ := strconv.ParseInt(r.FormValue("model_id"), 10, 64)
	key := cartKey(id, modelID)
	cart := readCart(r)
	delete(cart, key)
	writeCart(w, cart)
	http.Redirect(w, r, "/cart", http.StatusSeeOther)
}

func updateCartHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}
	id, _ := strconv.ParseInt(r.FormValue("product_id"), 10, 64)
	modelID, _ := strconv.ParseInt(r.FormValue("model_id"), 10, 64)
	qty, _ := strconv.Atoi(r.FormValue("qty"))
	key := cartKey(id, modelID)
	cart := readCart(r)
	if qty <= 0 {
		delete(cart, key)
	} else {
		cart[key] = qty
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
		productName := line.Product.Name
		if line.ModelName != "" {
			productName = productName + " (" + line.ModelName + ")"
		}
		_, _ = db.DB.Exec(`INSERT INTO order_items (order_id, product_id, product_name, brand, unit, qty, price_cents, line_cents) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			orderID, line.Product.ID, productName, line.Product.Brand, line.Product.Unit, line.Qty, line.Product.PriceCents, line.SubCents)
	}
	writeCart(w, map[string]int{})
	http.Redirect(w, r, "/track?hash="+hash+"&phone="+phone, http.StatusSeeOther)
}
