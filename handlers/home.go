package handlers

import (
	"net/http"

	"shoper/db"
	"shoper/models"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	hs := db.LoadHomeSettings()
	var products []models.Product
	if len(hs.RecommendIDs) > 0 {
		products, _ = db.ListProductsByIDs(hs.RecommendIDs)
	}
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	render(w, r, models.PageData{View: "home", Title: pn, Products: products, HomeSettings: hs})
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	title := db.GetSetting("about_title")
	if title == "" {
		title = "关于 " + pn
	}
	content := db.GetSetting("about_content")
	render(w, r, models.PageData{View: "about", Title: title + " - " + pn, AboutTitle: title, AboutContentRaw: content})
}
