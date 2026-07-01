package handlers

import (
	"net/http"
	"strings"

	"shoper/db"
	"shoper/middleware"
	"shoper/models"
)

func adminSettingsAccountHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	if r.Method == http.MethodPost {
		platformName := strings.TrimSpace(r.FormValue("platform_name"))
		platformSubtitle := strings.TrimSpace(r.FormValue("platform_subtitle"))
		newPhone := strings.TrimSpace(r.FormValue("admin_phone"))
		newKey := strings.TrimSpace(r.FormValue("admin_key"))
		if platformName != "" {
			db.SetSetting("platform_name", platformName)
		}
		if platformSubtitle != "" {
			db.SetSetting("platform_subtitle", platformSubtitle)
		}
		if newPhone != "" && newPhone != middleware.GetPhone() {
			db.SetSetting("admin_phone", newPhone)
			middleware.SetPhone(newPhone)
		}
		if newKey != "" && newKey != middleware.GetSecret() {
			db.SetSetting("admin_key", newKey)
			middleware.SetSecret(newKey)
			middleware.SetAdminCookie(w)
		}
		http.Redirect(w, r, "/admin/settings/account", http.StatusSeeOther)
		return
	}
	render(w, r, models.PageData{
		View:         "admin",
		AdminView:    "settings-account",
		Title:        "账户密码 - " + pn,
		PlatformName: pn,
		SettingsKey:  middleware.GetSecret(),
	})
}

func adminSettingsHomeHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	if r.Method == http.MethodPost {
		section := strings.TrimSpace(r.FormValue("section"))
		switch section {
		case "hero":
			db.SetSetting("home_hero_left_img", strings.TrimSpace(r.FormValue("home_hero_left_img")))
			db.SetSetting("home_hero_left_link", strings.TrimSpace(r.FormValue("home_hero_left_link")))
			db.SetSetting("home_hero_left_text", strings.TrimSpace(r.FormValue("home_hero_left_text")))
			db.SetSetting("home_hero_right_img", strings.TrimSpace(r.FormValue("home_hero_right_img")))
			db.SetSetting("home_hero_right_link", strings.TrimSpace(r.FormValue("home_hero_right_link")))
			db.SetSetting("home_hero_right_text", strings.TrimSpace(r.FormValue("home_hero_right_text")))
		case "carousel":
			db.SetSetting("home_carousel1", strings.TrimSpace(r.FormValue("home_carousel1")))
			db.SetSetting("home_carousel2", strings.TrimSpace(r.FormValue("home_carousel2")))
		case "recommend":
			db.SetSetting("home_recommend_ids", strings.TrimSpace(r.FormValue("home_recommend_ids")))
		default:
			db.SetSetting("home_hero_left_img", strings.TrimSpace(r.FormValue("home_hero_left_img")))
			db.SetSetting("home_hero_left_link", strings.TrimSpace(r.FormValue("home_hero_left_link")))
			db.SetSetting("home_hero_left_text", strings.TrimSpace(r.FormValue("home_hero_left_text")))
			db.SetSetting("home_hero_right_img", strings.TrimSpace(r.FormValue("home_hero_right_img")))
			db.SetSetting("home_hero_right_link", strings.TrimSpace(r.FormValue("home_hero_right_link")))
			db.SetSetting("home_hero_right_text", strings.TrimSpace(r.FormValue("home_hero_right_text")))
			db.SetSetting("home_carousel1", strings.TrimSpace(r.FormValue("home_carousel1")))
			db.SetSetting("home_carousel2", strings.TrimSpace(r.FormValue("home_carousel2")))
			db.SetSetting("home_recommend_ids", strings.TrimSpace(r.FormValue("home_recommend_ids")))
		}
		http.Redirect(w, r, "/admin/settings/home", http.StatusSeeOther)
		return
	}
	hs := db.LoadHomeSettings()
	var recommendProducts []models.Product
	if len(hs.RecommendIDs) > 0 {
		recommendProducts, _ = db.ListProductsByIDs(hs.RecommendIDs)
	}
	allProducts, _ := db.ListProducts("", "", "", 0)
	render(w, r, models.PageData{
		View:          "admin",
		AdminView:     "settings-home",
		Title:         "首页设置 - " + pn,
		PlatformName:  pn,
		HomeSettings:  hs,
		AllProducts:   allProducts,
		Products:      recommendProducts,
		AllCategories: db.LoadAllCategoriesWithTags(),
	})
}

func adminSettingsAboutHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	if r.Method == http.MethodPost {
		aboutTitle := strings.TrimSpace(r.FormValue("about_title"))
		aboutContent := strings.TrimSpace(r.FormValue("about_content"))
		db.SetSetting("about_title", aboutTitle)
		db.SetSetting("about_content", aboutContent)
		http.Redirect(w, r, "/admin/settings/about", http.StatusSeeOther)
		return
	}
	render(w, r, models.PageData{
		View:            "admin",
		AdminView:       "settings-about",
		Title:           "关于本店 - " + pn,
		PlatformName:    pn,
		AboutTitle:      db.GetSetting("about_title"),
		AboutContentRaw: db.GetSetting("about_content"),
	})
}

func adminSettingsTicketHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	if r.Method == http.MethodPost {
		db.SetSetting("ticket_qrcode", strings.TrimSpace(r.FormValue("ticket_qrcode")))
		db.SetSetting("ticket_contact", strings.TrimSpace(r.FormValue("ticket_contact")))
		db.SetSetting("ticket_contact2", strings.TrimSpace(r.FormValue("ticket_contact2")))
		db.SetSetting("ticket_description", strings.TrimSpace(r.FormValue("ticket_description")))
		db.SetSetting("ticket_trackqr", r.FormValue("ticket_trackqr"))
		http.Redirect(w, r, "/admin/settings/ticket", http.StatusSeeOther)
		return
	}
	render(w, r, models.PageData{
		View:         "admin",
		AdminView:    "settings-ticket",
		Title:        "票据设置 - " + pn,
		PlatformName: pn,
		Ticket:       db.LoadTicketSettings(),
	})
}

func adminSettingsFeatureHandler(w http.ResponseWriter, r *http.Request) {
	if !middleware.IsAdmin(r) {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodPost {
		cart := strings.TrimSpace(r.FormValue("feature_cart"))
		voucher := strings.TrimSpace(r.FormValue("feature_voucher"))
		order := strings.TrimSpace(r.FormValue("feature_order"))
		trade := strings.TrimSpace(r.FormValue("feature_trade"))
		if cart != "frontend" && cart != "backend" {
			cart = "frontend"
		}
		if voucher != "frontend" && voucher != "backend" {
			voucher = "frontend"
		}
		if order != "frontend" && order != "backend" {
			order = "frontend"
		}
		if trade != "enabled" && trade != "disabled" {
			trade = "enabled"
		}
		db.SetSetting("feature_cart", cart)
		db.SetSetting("feature_voucher", voucher)
		db.SetSetting("feature_order", order)
		db.SetSetting("feature_trade", trade)
		http.Redirect(w, r, "/admin/settings/account", http.StatusSeeOther)
		return
	}
	// 功能设置已并入基本设置页，直接跳转过去
	http.Redirect(w, r, "/admin/settings/account", http.StatusSeeOther)
}
