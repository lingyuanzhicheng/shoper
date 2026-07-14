package main

import (
	"context"
	"database/sql"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

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

//go:embed assets/favicon.ico
var faviconFS embed.FS

func main() {
	envUsername := strings.TrimSpace(os.Getenv("SHOPER_ADMIN_USERNAME"))
	envPassword := strings.TrimSpace(os.Getenv("SHOPER_ADMIN_PASSWORD"))
	if envUsername == "" || envPassword == "" {
		envUsername = utils.FirstNonEmpty(envUsername, "shoper")
		envPassword = utils.FirstNonEmpty(envPassword, "shoper")
		log.Println("SHOPER_ADMIN_USERNAME or SHOPER_ADMIN_PASSWORD is not set; using local development credentials")
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
	middleware.SetCredentials(envUsername, envPassword)

	// 如果数据库已有凭据，则用数据库的覆盖
	if v := db.GetSetting("admin_username"); v != "" {
		middleware.SetUsername(v)
	} else {
		db.SetSetting("admin_username", envUsername)
	}
	if v := db.GetSetting("admin_password"); v != "" {
		middleware.SetPassword(v)
	} else {
		db.SetSetting("admin_password", envPassword)
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
	mux.Handle("/uploads/product/", http.StripPrefix("/uploads/product/", http.FileServer(http.Dir("uploads/product"))))
	mux.Handle("/uploads/brand/", http.StripPrefix("/uploads/brand/", http.FileServer(http.Dir("uploads/brand"))))
	mux.Handle("/uploads/extra/", http.StripPrefix("/uploads/extra/", http.FileServer(http.Dir("uploads/extra"))))
	mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/x-icon")
		data, _ := faviconFS.ReadFile("assets/favicon.ico")
		w.Write(data)
	})
	handlers.RegisterRoutes(mux)

	srv := &http.Server{Addr: ":8080", Handler: mux}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Println("Shoper listening on http://localhost:8080")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen error: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shoper shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	if err := database.Close(); err != nil {
		log.Printf("database close error: %v", err)
	}
	log.Println("Shoper stopped")
}
