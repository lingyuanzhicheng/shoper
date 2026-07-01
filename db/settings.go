package db

import (
	"encoding/json"
	"shoper/models"
	"strconv"
	"strings"
)

func GetSetting(key string) string {
	var val string
	err := DB.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&val)
	if err != nil {
		return ""
	}
	return val
}

func SetSetting(key, value string) {
	_, _ = DB.Exec(`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
}

func LoadHomeSettings() models.HomeSettings {
	hs := models.HomeSettings{
		HeroLeftImg:   GetSetting("home_hero_left_img"),
		HeroLeftLink:  GetSetting("home_hero_left_link"),
		HeroLeftText:  GetSetting("home_hero_left_text"),
		HeroRightImg:  GetSetting("home_hero_right_img"),
		HeroRightLink: GetSetting("home_hero_right_link"),
		HeroRightText: GetSetting("home_hero_right_text"),
		Carousel1:     []models.CarouselSlide{},
		Carousel2:     []models.CarouselSlide{},
		RecommendIDs:  []int64{},
	}
	if raw := GetSetting("home_carousel1"); raw != "" && raw != "null" {
		json.Unmarshal([]byte(raw), &hs.Carousel1)
	}
	if raw := GetSetting("home_carousel2"); raw != "" && raw != "null" {
		json.Unmarshal([]byte(raw), &hs.Carousel2)
	}
	if raw := GetSetting("home_recommend_ids"); raw != "" && raw != "null" {
		json.Unmarshal([]byte(raw), &hs.RecommendIDs)
	}
	rc, _ := strconv.Atoi(strings.TrimSpace(GetSetting("home_recommend_count")))
	if rc < 1 {
		rc = 4
	}
	hs.RecommendCount = rc
	return hs
}

// LoadFeatureSettings 读取功能设置并计算综合可用性。
// 存储约定：
//   - feature_cart:    "frontend" 开启前台 / "backend" 仅限后台（默认 frontend）
//   - feature_voucher: "frontend" / "backend"（默认 frontend，暂未关联功能）
//   - feature_order:   "frontend" / "backend"（默认 frontend）
//   - feature_trade:   "enabled" 开启支持 / "disabled" 关闭支持（默认 enabled）
//
// 当 feature_trade 为 disabled 时，购物车、票据、订单均视为关闭（综合判定为 false）。
func LoadFeatureSettings() models.FeatureSettings {
	f := models.FeatureSettings{
		CartMode:     GetSetting("feature_cart"),
		VoucherMode:  GetSetting("feature_voucher"),
		OrderMode:    GetSetting("feature_order"),
		TradeEnabled: GetSetting("feature_trade") != "disabled",
	}
	if f.CartMode == "" {
		f.CartMode = "frontend"
	}
	if f.VoucherMode == "" {
		f.VoucherMode = "frontend"
	}
	if f.OrderMode == "" {
		f.OrderMode = "frontend"
	}
	f.CartEnabled = f.TradeEnabled && f.CartMode == "frontend"
	f.VoucherEnabled = f.TradeEnabled && f.VoucherMode == "frontend"
	f.OrderEnabled = f.TradeEnabled && f.OrderMode == "frontend"
	return f
}

// LoadTicketSettings 读取订单票据设置。
// 存储约定：
//   - ticket_qrcode:      二维码图片地址（上传至媒体管理的额外配图）
//   - ticket_contact:     平台联系方式1
//   - ticket_contact2:    平台联系方式2
//   - ticket_description: 票据说明
//   - ticket_trackqr:     订单追踪二维码开关 "1"/"0"
func LoadTicketSettings() models.TicketSettings {
	return models.TicketSettings{
		QRCodeURL:   GetSetting("ticket_qrcode"),
		Contact:     GetSetting("ticket_contact"),
		Contact2:    GetSetting("ticket_contact2"),
		Description: GetSetting("ticket_description"),
		TrackQR:     GetSetting("ticket_trackqr") == "1",
	}
}
