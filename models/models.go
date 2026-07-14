package models

import "html/template"

type Product struct {
	ID          int64
	Slug        string
	Group       string
	Name        string
	Brand       string
	BrandLogo   string
	Unit        string
	PriceCents  int64
	Description string
	Images      []string
	CartQty     int
	CategoryID  int64
	Body        string
	Tags        []Tag
	Models      []ProductModel
}

type ProductModel struct {
	ID         int64  `json:"id"`
	ProductID  int64  `json:"product_id"`
	ModelName  string `json:"name"`
	PriceCents int64  `json:"price_cents"`
	Unit       string `json:"unit"`
	SortOrder  int    `json:"sort_order"`
}

type Category struct {
	ID        int64
	Name      string
	URL       string
	Icon      string
	ParentID  *int64
	SortOrder int
	Children  []Category
	Tags      []Tag
}

type Tag struct {
	ID         int64
	Name       string
	CategoryID int64
	SortOrder  int
}

type Brand struct {
	ID          int64
	Name        string
	Logo        string
	Description string
	SortOrder   int
}

type CartLine struct {
	Product   Product
	Qty       int
	SubCents  int64
	ModelID   int64
	ModelName string
}

type Order struct {
	ID            int64
	Hash          string
	ContactName   string
	Phone         string
	Address       string
	Community     string
	Building      string
	UnitNo        string
	Room          string
	Notes         string
	Status        string
	Archived      bool
	DiscountCents int64
	PaidCents     int64
	TotalCents    int64
	CreatedAt     string
	Items         []OrderItem
	Vouchers      []Voucher
}

type Voucher struct {
	ID          int64
	OrderHash   string
	Type        string
	ImageURL    string
	Description string
	AmountCents int64
	IsRefund    bool
	CreatedAt   string
}

type OrderItem struct {
	ID          int64
	ProductID   int64
	ProductName string
	Brand       string
	Unit        string
	Qty         int
	PriceCents  int64
	LineCents   int64
	Delivered   bool
	IsReturn    bool
	Image       string
}

type Pagination struct {
	Page     int
	Size     int
	Total    int
	Pages    int
	HasPrev  bool
	HasNext  bool
	PrevPage int
	NextPage int
	BaseURL  string
	RawQuery string
}

type PageData struct {
	Title       string
	View        string
	AdminView   string
	Query       string
	Group       string
	Brand       string
	Message     string
	MessageType string
	IsAdmin     bool
	AdminPhone  string
	Products    []Product
	Product     Product
	Categories  []Category
	CartLines   []CartLine
	CartCount   int
	CartTotal   int64
	Groups      []string
	Order       Order
	Orders      []Order
	CurrentYear int
	Pagination  Pagination

	AllProducts      []Product
	AllBrands        []Brand
	EditProduct      Product
	AllCategories    []Category
	ParentCategories []Category
	ProductBody      template.HTML
	DescriptionHTML  template.HTML
	ProductBodyRaw   string
	DescriptionRaw   string
	CategoryID       int64
	CategoryName     string
	TagID            int64
	TagIDs           map[int64]bool
	Tags             []Tag

	Community string
	Building  string
	UnitNo    string
	Room      string
	Notes     string

	PlatformName     string
	PlatformSubtitle string
	AboutTitle       string
	AboutContent     template.HTML
	AboutContentRaw  string
	SettingsKey      string
	HomeSettings     HomeSettings
	MediaFiles       []MediaFile
	Feature          FeatureSettings
	Ticket           TicketSettings
}

// FeatureSettings 描述平台功能设置开关的当前状态。
// 各字段含义见 db.LoadFeatureSettings。
type FeatureSettings struct {
	// CartMode 购物车功能开关："frontend"（开启前台）/ "backend"（仅限后台）
	CartMode string
	// VoucherMode 票据功能开关："frontend" / "backend"（暂未做功能关联）
	VoucherMode string
	// OrderMode 订单功能开关："frontend"（前台购物车显示提交订单）/ "backend"（仅限后台）
	OrderMode string
	// TradeEnabled 交易功能开关：true=开启支持，false=关闭支持（一键关闭所有）
	TradeEnabled bool

	// CartEnabled 综合判定后的购物车是否可用（前台可见）
	CartEnabled bool
	// OrderEnabled 综合判定后的提交订单是否可用（前台可见）
	OrderEnabled bool
	// VoucherEnabled 综合判定后的票据是否可用（前台可见）
	VoucherEnabled bool
}

// TicketSettings 描述订单票据的配置内容。
// QRCodeURL 为票据设置上传的二维码图片地址（存储于媒体管理的额外配图）；
// Contact/Contact2 为平台联系方式1/2；Description 为票据说明；
// TrackQR 控制是否在票据上生成订单追踪二维码。
type TicketSettings struct {
	QRCodeURL   string
	Contact     string
	Contact2    string
	Description string
	TrackQR     bool
}

type MediaFile struct {
	Filename string
	URL      string
	Type     string // "product" "brand" "extra"
	TypeName string // "商品配图" "品牌图标" "额外配图"
}

type CarouselSlide struct {
	Img  string `json:"img"`
	Text string `json:"text"`
	Link string `json:"link"`
}

type HomeSettings struct {
	HeroLeftImg    string
	HeroLeftLink   string
	HeroLeftText   string
	HeroRightImg   string
	HeroRightLink  string
	HeroRightText  string
	Carousel1      []CarouselSlide
	Carousel2      []CarouselSlide
	RecommendIDs   []int64
	RecommendCount int
}
