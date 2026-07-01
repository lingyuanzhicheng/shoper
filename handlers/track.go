package handlers

import (
	"net/http"
	"strings"

	"shoper/db"
	"shoper/middleware"
	"shoper/models"
)

func trackHandler(w http.ResponseWriter, r *http.Request) {
	feature := db.LoadFeatureSettings()
	if !feature.TradeEnabled && !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodPost {
		http.Redirect(w, r, "/track?hash="+r.FormValue("hash")+"&phone="+r.FormValue("phone"), http.StatusSeeOther)
		return
	}
	hash := strings.TrimSpace(r.URL.Query().Get("hash"))
	phone := strings.TrimSpace(r.URL.Query().Get("phone"))
	data := models.PageData{View: "track", Title: "订单追踪 - Shoper", Query: hash}
	if hash != "" && phone != "" {
		order, err := db.GetOrderByHashForCustomer(hash)
		if err != nil {
			data.Message = "订单不存在"
			data.MessageType = "error"
		} else if order.Phone != phone {
			data.Message = "订单号或手机号不匹配"
			data.MessageType = "error"
		} else if order.Archived {
			data.Message = "此订单已被归档"
			data.MessageType = "error"
		} else if order.Status == "已取消" {
			data.Message = "此订单已被取消"
			data.MessageType = "error"
		} else {
			order.Vouchers, _ = db.GetOrderVouchers(hash)
			data.Order = order
		}
	}
	render(w, r, data)
}
