package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"shoper/db"
	"shoper/middleware"
	"shoper/models"
)

// adminOrderTicketExportHandler 后台订单详情页导出订单票据图片。
func adminOrderTicketExportHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	feature := db.LoadFeatureSettings()
	if !feature.TradeEnabled {
		http.Error(w, "交易功能已关闭", http.StatusForbidden)
		return
	}
	hash := strings.TrimPrefix(r.URL.Path, "/admin/order/ticket/export/")
	if hash == "" {
		http.Redirect(w, r, "/admin/orders", http.StatusSeeOther)
		return
	}
	order, err := db.GetOrderByHash(hash)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}
	order.Vouchers, _ = db.GetOrderVouchers(hash)
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	ts := db.LoadTicketSettings()
	trackURL := "http://" + r.Host + "/track?hash=" + order.Hash + "&phone=" + order.Phone
	imgBytes, err := RenderTicket(order, ts, pn, trackURL, false)
	if err != nil {
		http.Error(w, "render ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=order-ticket-%s.jpg", hash))
	w.Write(imgBytes)
}

// cartTicketExportHandler 前台购物车导出票据图片（基于当前购物车内容）。
// 购物车票据按钮需购物车与票据均开启前台；管理员登录时只要交易开启即可。
func cartTicketExportHandler(w http.ResponseWriter, r *http.Request) {
	feature := db.LoadFeatureSettings()
	if !feature.TradeEnabled {
		http.Error(w, "交易功能已关闭", http.StatusForbidden)
		return
	}
	isAdmin := middleware.IsAdmin(r)
	if (!feature.CartEnabled || !feature.VoucherEnabled) && !isAdmin {
		http.Error(w, "票据功能未开放", http.StatusForbidden)
		return
	}
	lines, total, err := cartLines(r)
	if err != nil || len(lines) == 0 {
		http.Redirect(w, r, "/cart", http.StatusSeeOther)
		return
	}
	var items []models.OrderItem
	for _, line := range lines {
		productName := line.Product.Name
		if line.ModelName != "" {
			productName = productName + " (" + line.ModelName + ")"
		}
		items = append(items, models.OrderItem{
			ProductName: productName,
			Brand:       line.Product.Brand,
			Unit:        line.Product.Unit,
			Qty:         line.Qty,
			PriceCents:  line.Product.PriceCents,
			LineCents:   line.SubCents,
		})
	}
	order := models.Order{
		Hash:         "CART-" + time.Now().Format("0102150405"),
		ContactName:  "（待提交）",
		Phone:        "（待提交）",
		Address:      "（待提交）",
		TotalCents:   total,
		DiscountCents: 0,
		PaidCents:    0,
		Status:       "待确认",
		CreatedAt:    time.Now().Format("2006-01-02 15:04"),
		Items:        items,
	}
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	ts := db.LoadTicketSettings()
	// 购物车票据为报价票据单
	imgBytes, err := RenderTicket(order, ts, pn, "", true)
	if err != nil {
		http.Error(w, "render ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", "attachment; filename=quote-ticket.jpg")
	w.Write(imgBytes)
}

// trackTicketExportHandler 订单追踪页导出票据图片。
// 非管理员需票据功能开启前台；管理员登录时只要交易开启即可。
func trackTicketExportHandler(w http.ResponseWriter, r *http.Request) {
	feature := db.LoadFeatureSettings()
	if !feature.TradeEnabled {
		http.Error(w, "交易功能已关闭", http.StatusForbidden)
		return
	}
	isAdmin := middleware.IsAdmin(r)
	if !feature.VoucherEnabled && !isAdmin {
		http.Error(w, "票据功能未开放", http.StatusForbidden)
		return
	}
	hash := strings.TrimSpace(r.URL.Query().Get("hash"))
	phone := strings.TrimSpace(r.URL.Query().Get("phone"))
	if hash == "" {
		http.Redirect(w, r, "/track", http.StatusSeeOther)
		return
	}
	order, err := db.GetOrderByHashForCustomer(hash)
	if err != nil {
		http.Error(w, "order not found", http.StatusNotFound)
		return
	}
	// 非管理员需校验手机号
	if !isAdmin {
		if order.Phone != phone {
			http.Error(w, "订单号或手机号不匹配", http.StatusForbidden)
			return
		}
		if order.Archived || order.Status == "已取消" {
			http.Error(w, "订单不可用", http.StatusForbidden)
			return
		}
	}
	order.Vouchers, _ = db.GetOrderVouchers(hash)
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	ts := db.LoadTicketSettings()
	trackURL := "http://" + r.Host + "/track?hash=" + order.Hash + "&phone=" + order.Phone
	imgBytes, err := RenderTicket(order, ts, pn, trackURL, false)
	if err != nil {
		http.Error(w, "render ticket: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=order-ticket-%s.jpg", hash))
	w.Write(imgBytes)
}
