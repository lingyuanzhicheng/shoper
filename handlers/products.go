package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"shoper/db"
	"shoper/models"
)

func parseInt64(s string) (int64, error) {
	return strconv.ParseInt(s, 10, 64)
}

func productsHandler(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	group := strings.TrimSpace(r.URL.Query().Get("group"))
	brand := strings.TrimSpace(r.URL.Query().Get("brand"))
	catID, _ := parseInt64(r.URL.Query().Get("cat"))
	tagID, _ := parseInt64(r.URL.Query().Get("tag"))

	catName := ""
	var tags []models.Tag
	if catID > 0 {
		if cat, err := db.GetCategoryByID(catID); err == nil {
			catName = cat.Name
			group = cat.Name
		}
		tags, _ = db.GetTagsByCategory(catID)
	} else {
		cats, _ := db.GetAllCategories()
		for _, c := range cats {
			t, _ := db.GetTagsByCategory(c.ID)
			tags = append(tags, t...)
		}
	}

	page, size := parsePage(r)
	products, total, err := db.ListProductsPaged(q, group, brand, page, size)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	categories, _ := db.GetAllCategories()
	title := "商品列表 - Shoper"
	if catName != "" {
		title = catName + " - Shoper"
	}
	render(w, r, models.PageData{View: "products", Title: title, Query: q, Group: group, Brand: brand, Products: products, AllCategories: categories, CategoryID: catID, CategoryName: catName, Tags: tags, TagID: tagID, Pagination: buildPagination("/products", filterQuery(r), page, size, total)})
}

func brandsHandler(w http.ResponseWriter, r *http.Request) {
	pn := db.GetSetting("platform_name")
	if pn == "" {
		pn = "Shoper"
	}
	page, size := parsePage(r)
	brands, total, err := db.GetAllBrandsPaged(page, size)
	if err != nil {
		brands, _ = db.GetAllBrands()
		total = len(brands)
	}
	if brands == nil {
		brands = []models.Brand{}
	}
	allProducts, _ := db.ListProducts("", "", "", 0)
	if allProducts == nil {
		allProducts = []models.Product{}
	}
	for i := range allProducts {
		allProducts[i].Images, _ = db.ProductImages(allProducts[i].ID)
	}
	allCats := db.LoadAllCategoriesWithTags()
	if allCats == nil {
		allCats = []models.Category{}
	}
	render(w, r, models.PageData{
		View:          "brands",
		Title:         "品牌列表 - " + pn,
		AllBrands:     brands,
		AllProducts:   allProducts,
		AllCategories: allCats,
		Pagination:    buildPagination("/brands", filterQuery(r), page, size, total),
	})
}

func productHandler(w http.ResponseWriter, r *http.Request) {
	slug := strings.TrimPrefix(r.URL.Path, "/product/")
	product, err := db.GetProductBySlug(slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	message := ""
	if r.URL.Query().Get("added") == "1" {
		message = "已添加到购物车，请前往购物车提交订单"
	}
	product.CartQty = readCart(r)[product.ID]

	render(w, r, models.PageData{View: "product", Title: product.Name + " - Shoper", Product: product, Message: message, MessageType: "success", ProductBodyRaw: product.Body, DescriptionRaw: product.Description, Tags: product.Tags})
}
