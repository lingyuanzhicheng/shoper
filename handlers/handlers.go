package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"shoper/db"
	"shoper/middleware"
	"shoper/models"
	"shoper/utils"
)

var tmpl *template.Template

func InitTemplates(t *template.Template) {
	tmpl = t
}

func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"price":  utils.FormatPrice,
		"mul":    func(a, b int64) int64 { return a * b },
		"unsafe": func(s string) template.HTML { return template.HTML(s) },
		"json": func(v any) template.JS {
			b, _ := json.Marshal(v)
			return template.JS(b)
		},
		"add": func(a, b int64) int64 { return a + b },
		"sub": func(a, b int64) int64 { return a - b },
		"formatDate": func(s string) string {
			if s == "" {
				return ""
			}
			if len(s) >= 10 {
				s = s[:10]
			}
			if len(s) == 10 && s[4] == '-' {
				return s[:4] + "/" + s[5:7] + "/" + s[8:]
			}
			return s
		},
		"formatTime": func(s string) string {
			if s == "" {
				return ""
			}
			if len(s) >= 19 {
				return s[11:19]
			}
			if len(s) >= 16 {
				return s[11:16] + ":00"
			}
			return ""
		},
		"formatPrice": utils.FormatPrice,
		"pageURL":     pageURL,
		"sizeURL":     sizeURL,
		"contentTemplate": func(view, adminView string) string {
			if view == "admin" && adminView != "" {
				return "content_" + view + "_" + adminView
			}
			return "content_" + view
		},
	}
}

func render(w http.ResponseWriter, r *http.Request, data models.PageData) {
	allCats, _ := db.GetAllCategories()
	children := make([]models.Category, 0, len(allCats)+1)
	children = append(children, models.Category{Name: "全部商品", URL: "/products"})
	for _, c := range allCats {
		children = append(children, models.Category{Name: c.Name, URL: "/products?cat=" + strconv.FormatInt(c.ID, 10)})
	}
	data.Categories = []models.Category{
		{Name: "首页", URL: "/", Icon: "sui-Structuresquarescontrol"},
		{Name: "商品列表", URL: "/products", Icon: "sui-Storeshope-commerce", Children: children},
		{Name: "品牌列表", URL: "/brands", Icon: "sui-Taglabele-commerce"},
	}
	// 订单追踪导航：交易关闭时隐藏（管理员仍可见）
	if data.Feature.TradeEnabled || data.IsAdmin {
		data.Categories = append(data.Categories, models.Category{Name: "订单追踪", URL: "/track", Icon: "sui-Magnifiercontrol"})
	}
	// 购物车导航：交易关闭时对所有人隐藏；仅限后台时仅管理员可见；开启前台时所有人可见
	if data.Feature.TradeEnabled && (data.Feature.CartEnabled || data.IsAdmin) {
		data.Categories = append(data.Categories, models.Category{Name: "购物车", URL: "/cart", Icon: "sui-Shoppingcart"})
	}
	data.Categories = append(data.Categories, models.Category{Name: "关于本店", URL: "/about", Icon: "sui-Bookmarks"})
	var groupNames []string
	for _, c := range allCats {
		groupNames = append(groupNames, c.Name)
	}
	data.Groups = groupNames
	data.CartCount = cartCount(r)
	data.CurrentYear = time.Now().Year()
	data.IsAdmin = middleware.IsAdmin(r)
	data.AdminPhone = middleware.GetPhone()
	data.PlatformName = db.GetSetting("platform_name")
	if data.PlatformName == "" {
		data.PlatformName = "Shoper"
	}
	data.PlatformSubtitle = db.GetSetting("platform_subtitle")
	data.Feature = db.LoadFeatureSettings()
	data.Ticket = db.LoadTicketSettings()
	brands, _ := db.GetAllBrands()
	data.AllBrands = brands
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "shop.html", data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

const defaultPageSize = 30

func parsePage(r *http.Request) (int, int) {
	size := defaultPageSize
	if s := r.URL.Query().Get("size"); s != "" {
		if v, err := strconv.Atoi(s); err == nil && (v == 30 || v == 50 || v == 100) {
			size = v
		}
	}
	page := 1
	if p := r.URL.Query().Get("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	return page, size
}

// filterQuery returns the URL query string with page/size removed, so the
// pagination control can re-append its own page/size while preserving filters.
func filterQuery(r *http.Request) string {
	q := r.URL.Query()
	q.Del("page")
	q.Del("size")
	return q.Encode()
}

// buildPagination assembles a Pagination struct from total count and the
// current page/size. baseURL is the page path (e.g. "/products"), rawQuery
// carries preserved filter params (already stripped of page/size).
func buildPagination(baseURL, rawQuery string, page, size, total int) models.Pagination {
	pages := 0
	if size > 0 {
		pages = (total + size - 1) / size
	}
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	return models.Pagination{
		Page:     page,
		Size:     size,
		Total:    total,
		Pages:    pages,
		HasPrev:  page > 1,
		HasNext:  page < pages,
		PrevPage: page - 1,
		NextPage: page + 1,
		BaseURL:  baseURL,
		RawQuery: rawQuery,
	}
}

// pageURL is a template helper that builds a pagination link URL from a
// Pagination struct and a target page number, preserving filters & size.
func pageURL(p models.Pagination, page int) string {
	q := p.RawQuery
	if q != "" {
		q += "&"
	}
	q += "page=" + strconv.Itoa(page) + "&size=" + strconv.Itoa(p.Size)
	return p.BaseURL + "?" + q
}

// sizeURL builds a URL that switches the page size (resets to page 1),
// preserving filter params.
func sizeURL(p models.Pagination, size int) string {
	q := p.RawQuery
	if q != "" {
		q += "&"
	}
	q += "page=1&size=" + strconv.Itoa(size)
	return p.BaseURL + "?" + q
}

func RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/products", productsHandler)
	mux.HandleFunc("/brands", brandsHandler)
	mux.HandleFunc("/about", aboutHandler)
	mux.HandleFunc("/product/", productHandler)
	mux.HandleFunc("/cart", cartHandler)
	mux.HandleFunc("/cart/add", addCartHandler)
	mux.HandleFunc("/cart/update", updateCartHandler)
	mux.HandleFunc("/cart/remove", removeCartHandler)
	mux.HandleFunc("/order/create", createOrderHandler)
	mux.HandleFunc("/track", trackHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/logout", logoutHandler)
	mux.HandleFunc("/admin/orders", adminOrdersHandler)
	mux.HandleFunc("/admin/orders/", adminOrderDetailHandler)
	mux.HandleFunc("/admin/order/delete/", adminOrderDeleteHandler)
	mux.HandleFunc("/admin/order/archive/", adminOrderArchiveHandler)
	mux.HandleFunc("/admin/order/update/", adminOrderUpdateHandler)
	mux.HandleFunc("/admin/order/customer/update/", adminOrderCustomerUpdateHandler)
	mux.HandleFunc("/admin/order/status/update/", adminOrderStatusUpdateHandler)
	mux.HandleFunc("/admin/order/voucher/add", adminOrderVoucherAddHandler)
	mux.HandleFunc("/admin/order/voucher/delete", adminOrderVoucherDeleteHandler)
	mux.HandleFunc("/admin/products", adminProductsHandler)
	mux.HandleFunc("/admin/product/new", adminProductNewHandler)
	mux.HandleFunc("/admin/product/", adminProductEditHandler)
	mux.HandleFunc("/admin/tags/by-category", adminTagsByCategoryHandler)
	mux.HandleFunc("/admin/categories", adminCategoriesHandler)
	mux.HandleFunc("/admin/brands", adminBrandsHandler)
	mux.HandleFunc("/admin/media", adminMediaHandler)
	mux.HandleFunc("/admin/settings/account", adminSettingsAccountHandler)
	mux.HandleFunc("/admin/settings/home", adminSettingsHomeHandler)
	mux.HandleFunc("/admin/settings/about", adminSettingsAboutHandler)
	mux.HandleFunc("/admin/settings/feature", adminSettingsFeatureHandler)
	mux.HandleFunc("/admin/settings/ticket", adminSettingsTicketHandler)
	mux.HandleFunc("/admin/order/ticket/export/", adminOrderTicketExportHandler)
	mux.HandleFunc("/cart/ticket/export", cartTicketExportHandler)
	mux.HandleFunc("/track/ticket/export", trackTicketExportHandler)
	mux.HandleFunc("/upload", uploadHandler)
}
