package main

import (
	"database/sql"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"shoper/db"
	"shoper/handlers"
	"shoper/middleware"
	"shoper/utils"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed templates/*
var templateFS embed.FS

//go:embed static/*
var staticFS embed.FS

//go:embed assets/NotoSansSC.otf
//go:embed assets/MaShanZheng.ttf
var ticketFontFS embed.FS

func main() {
	envPhone := strings.TrimSpace(os.Getenv("SHOPER_ADMIN_PHONE"))
	envKey := strings.TrimSpace(os.Getenv("SHOPER_ADMIN_KEY"))
	if envPhone == "" || envKey == "" {
		envPhone = utils.FirstNonEmpty(envPhone, "shoper")
		envKey = utils.FirstNonEmpty(envKey, "shoper")
		log.Println("SHOPER_ADMIN_PHONE or SHOPER_ADMIN_KEY is not set; using local development credentials")
	}

	t, err := template.New("shop.html").Funcs(handlers.TemplateFuncs()).ParseFS(templateFS,
		"templates/shop.html",
		"templates/pages/*.html",
		"templates/pages/admin/*.html",
	)
	if err != nil {
		log.Fatal("parse templates:", err)
	}
	handlers.InitTemplates(t)

	if err := os.MkdirAll("data", 0755); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll("uploads", 0755); err != nil {
		log.Fatal(err)
	}
	database, err := sql.Open("sqlite3", filepath.Join("data", "shoper.db")+"?_foreign_keys=on")
	if err != nil {
		log.Fatal(err)
	}
	if err := db.InitDB(database); err != nil {
		log.Fatal(err)
	}

	// 先用环境变量/默认值初始化凭据
	middleware.SetCredentials(envPhone, envKey)

	// 如果数据库已有凭据，则用数据库的覆盖
	if v := db.GetSetting("admin_phone"); v != "" {
		middleware.SetPhone(v)
	} else {
		db.SetSetting("admin_phone", envPhone)
	}
	if v := db.GetSetting("admin_key"); v != "" {
		middleware.SetSecret(v)
	} else {
		db.SetSetting("admin_key", envKey)
	}
	if db.GetSetting("platform_name") == "" {
		db.SetSetting("platform_name", "Shoper")
	}
	if db.GetSetting("about_title") == "" {
		db.SetSetting("about_title", "关于 Shoper")
	}
	if db.GetSetting("about_content") == "" {
		db.SetSetting("about_content", "Shoper 是一家专注于空间材料与家居产品的商品目录平台。我们精选地面材料、墙面装饰、灯光照明、家具收纳与五金配件等品类，为客户提供清晰的产品信息与便捷的询价、下单体验。\n\n当前平台采用订单制管理：提交购物车后生成订单号，凭订单号和联系电话即可追踪订单状态。如需进一步咨询或定制方案，请联系客服。\n\n我们的产品适用于住宅、商业空间与设计师项目，支持按分类、品牌、关键词精准筛选，帮助您快速找到合适的产品。")
	}

	// 注册票据渲染字体
	if fontBytes, err := ticketFontFS.ReadFile("assets/NotoSansSC.otf"); err == nil {
		handlers.RegisterTicketFont(fontBytes)
	}
	if maBytes, err := ticketFontFS.ReadFile("assets/MaShanZheng.ttf"); err == nil {
		handlers.RegisterTicketBrandFont(maBytes)
	}

	mux := http.NewServeMux()
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))
	handlers.RegisterRoutes(mux)

	log.Println("Shoper listening on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
