package handlers

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
	"strings"

	"github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"shoper/models"
)

// ticketFontBytes 为正文字体（NotoSansSC），ticketBrandFontBytes 为平台名称字体（Ma Shan Zheng）。
var (
	ticketFontBytes     []byte
	ticketBrandFontBytes []byte
)

// RegisterTicketFont 注入用于票据渲染的正文字体数据。
func RegisterTicketFont(b []byte) { ticketFontBytes = b }

// RegisterTicketBrandFont 注入用于票据平台名称的字体数据。
func RegisterTicketBrandFont(b []byte) { ticketBrandFontBytes = b }

type ticketDrawer struct {
	img       *image.RGBA
	face      font.Face // 正文 18px
	faceSmall font.Face // 小字 16px
	faceBold  font.Face // 加粗 20px
	faceTitle font.Face // 订单票据单标题 40px
	faceBrand font.Face // 平台名称字体（Ma Shan Zheng）48px
	white     color.Color
	black     color.Color
	gray      color.Color
	light     color.Color
	line      color.Color
	border    color.Color
	red       color.Color
}

// toYuan 金额分转字符串（元，两位小数）。
func toYuan(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	return fmt.Sprintf("%s%d.%02d", sign, cents/100, cents%100)
}

// chineseCapital 金额分转中文大写。
func chineseCapital(cents int64) string {
	if cents == 0 {
		return "零元整"
	}
	neg := false
	if cents < 0 {
		neg = true
		cents = -cents
	}
	yuan := cents / 100
	jiao := (cents % 100) / 10
	fen := cents % 10
	digits := []string{"零", "壹", "贰", "叁", "肆", "伍", "陆", "柒", "捌", "玖"}
	units := []string{"", "拾", "佰", "仟", "万", "拾", "佰", "仟", "亿", "拾", "佰", "仟", "万"}
	var b strings.Builder
	if neg {
		b.WriteString("负")
	}
	s := fmt.Sprintf("%d", yuan)
	for i, c := range s {
		d := int(c - '0')
		pos := len(s) - i - 1
		if d == 0 {
			if pos%4 == 0 {
				b.WriteString(units[pos])
			} else {
				b.WriteString("零")
			}
		} else {
			b.WriteString(digits[d])
			b.WriteString(units[pos])
		}
	}
	res := b.String()
	for strings.Contains(res, "零零") {
		res = strings.ReplaceAll(res, "零零", "零")
	}
	res = strings.TrimRight(res, "零")
	b.Reset()
	b.WriteString(res)
	b.WriteString("元")
	if jiao == 0 && fen == 0 {
		b.WriteString("整")
	} else {
		if jiao == 0 {
			b.WriteString("零")
		} else {
			b.WriteString(digits[jiao])
			b.WriteString("角")
		}
		if fen != 0 {
			b.WriteString(digits[fen])
			b.WriteString("分")
		}
	}
	return b.String()
}

func newTicketDrawer(width, height int) (*ticketDrawer, error) {
	if len(ticketFontBytes) == 0 {
		return nil, fmt.Errorf("ticket font not registered")
	}
	parsed, err := opentype.Parse(ticketFontBytes)
	if err != nil {
		return nil, err
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 18, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	faceSmall, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 15, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		return nil, err
	}
	faceBold, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 19, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	faceTitle, err := opentype.NewFace(parsed, &opentype.FaceOptions{Size: 38, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, err
	}
	var faceBrand font.Face
	if len(ticketBrandFontBytes) > 0 {
		if bp, err := opentype.Parse(ticketBrandFontBytes); err == nil {
			faceBrand, _ = opentype.NewFace(bp, &opentype.FaceOptions{Size: 69, DPI: 72, Hinting: font.HintingFull})
		}
	}
	if faceBrand == nil {
		faceBrand = faceTitle
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	return &ticketDrawer{
		img:       img,
		face:      face,
		faceSmall: faceSmall,
		faceBold:  faceBold,
		faceTitle: faceTitle,
		faceBrand: faceBrand,
		white:     color.RGBA{255, 255, 255, 255},
		black:     color.RGBA{30, 41, 59, 255},
		gray:      color.RGBA{100, 116, 139, 255},
		light:     color.RGBA{241, 245, 249, 255},
		line:      color.RGBA{203, 213, 225, 255},
		border:    color.RGBA{71, 85, 105, 255},
		red:       color.RGBA{220, 38, 38, 255},
	}, nil
}

func (d *ticketDrawer) text(s string, x, y int, c color.Color, face font.Face) {
	(&font.Drawer{
		Dst:  d.img,
		Src:  image.NewUniform(c),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)},
	}).DrawString(s)
}

func (d *ticketDrawer) textRight(s string, rightX, y int, c color.Color, face font.Face) {
	w := font.MeasureString(face, s).Ceil()
	d.text(s, rightX-w, y, c, face)
}

func (d *ticketDrawer) textCenter(s string, cx, y int, c color.Color, face font.Face) {
	w := font.MeasureString(face, s).Ceil()
	d.text(s, cx-w/2, y, c, face)
}

func (d *ticketDrawer) fillRect(x1, y1, x2, y2 int, c color.Color) {
	r := image.Rect(x1, y1, x2, y2)
	draw.Draw(d.img, r, &image.Uniform{c}, image.Point{}, draw.Src)
}

func (d *ticketDrawer) hline(x1, x2, y int, c color.Color) {
	for x := x1; x < x2; x++ {
		d.img.Set(x, y, c)
	}
}

func (d *ticketDrawer) vline(x, y1, y2 int, c color.Color) {
	for y := y1; y < y2; y++ {
		d.img.Set(x, y, c)
	}
}

// drawCellBorder 绘制单元格四边框（thin）。
func (d *ticketDrawer) cellBox(x1, y1, x2, y2 int) {
	d.rect(x1, y1, x2, y2, d.border)
}

func (d *ticketDrawer) rect(x1, y1, x2, y2 int, c color.Color) {
	d.hline(x1, x2, y1, c)
	d.hline(x1, x2, y2-1, c)
	d.vline(x1, y1, y2, c)
	d.vline(x2-1, y1, y2, c)
}

// drawQRCode 在指定区域绘制二维码图片（白色背景，等比缩放居中）。
func (d *ticketDrawer) drawQRCode(path string, x, y, size int) {
	if path == "" {
		return
	}
	// 白色背景
	d.fillRect(x, y, x+size, y+size, d.white)
	abs := strings.TrimPrefix(path, "/uploads/")
	full := "uploads/" + abs
	f, err := os.Open(full)
	if err != nil {
		return
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return
	}
	b := img.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()
	if srcW == 0 || srcH == 0 {
		return
	}
	// 留 8px 白边
	pad := 8
	inner := size - pad*2
	scale := math.Min(float64(inner)/float64(srcW), float64(inner)/float64(srcH))
	newW := int(float64(srcW) * scale)
	newH := int(float64(srcH) * scale)
	ox := x + (size-newW)/2
	oy := y + (size-newH)/2
	for dy := 0; dy < newH; dy++ {
		for dx := 0; dx < newW; dx++ {
			sx := b.Min.X + int(float64(dx)/scale)
			sy := b.Min.Y + int(float64(dy)/scale)
			d.img.Set(ox+dx, oy+dy, img.At(sx, sy))
		}
	}
}

// drawQRCodeImage 在指定区域绘制已解码的二维码图片（白色背景，等比缩放居中）。
func (d *ticketDrawer) drawQRCodeImage(img image.Image, x, y, size int) {
	if img == nil {
		return
	}
	// 白色背景
	d.fillRect(x, y, x+size, y+size, d.white)
	b := img.Bounds()
	srcW := b.Dx()
	srcH := b.Dy()
	if srcW == 0 || srcH == 0 {
		return
	}
	pad := 8
	inner := size - pad*2
	scale := math.Min(float64(inner)/float64(srcW), float64(inner)/float64(srcH))
	newW := int(float64(srcW) * scale)
	newH := int(float64(srcH) * scale)
	ox := x + (size-newW)/2
	oy := y + (size-newH)/2
	for dy := 0; dy < newH; dy++ {
		for dx := 0; dx < newW; dx++ {
			sx := b.Min.X + int(float64(dx)/scale)
			sy := b.Min.Y + int(float64(dy)/scale)
			d.img.Set(ox+dx, oy+dy, img.At(sx, sy))
		}
	}
}

// generateTrackQRCode 生成订单追踪链接的二维码图片。
func generateTrackQRCode(trackURL string) image.Image {
	qr, err := qrcode.New(trackURL, qrcode.Medium)
	if err != nil {
		return nil
	}
	qr.DisableBorder = true
	return qr.Image(256)
}

// wrapText 按最大宽度换行文本。
func (d *ticketDrawer) wrapText(s string, maxWidth int, face font.Face) []string {
	if s == "" {
		return []string{}
	}
	runes := []rune(s)
	var lines []string
	var cur strings.Builder
	curW := 0
	for _, r := range runes {
		ch := string(r)
		cw := font.MeasureString(face, ch).Ceil()
		if curW+cw > maxWidth && cur.Len() > 0 {
			lines = append(lines, cur.String())
			cur.Reset()
			curW = 0
		}
		cur.WriteString(ch)
		curW += cw
	}
	if cur.Len() > 0 {
		lines = append(lines, cur.String())
	}
	return lines
}

// RenderTicket 根据订单与票据设置生成票据图片（JPEG 字节）。
// trackURL 为订单追踪链接，当 ts.TrackQR 为 true 时在票据右上角生成追踪二维码。
// isQuote 为 true 时生成报价票据单（购物车用）：无单号、无客户信息、无追踪二维码、
// 无抹零/应付/备注/待付/结账方式/确认，仅保留合计行与说明。
// 布局严格参照 shoper.xlsx 票据案例。
func RenderTicket(order models.Order, ts models.TicketSettings, platformName, trackURL string, isQuote bool) ([]byte, error) {
	// 列宽按比例：序号0.75、货号及品名4、单位1、数量1、单价1.25、金额2.25、备注2.75
	// 总比例 13，表格宽度 881，1单位≈67.8px
	const (
		W = 1000
		ml = 60              // 左边距（外边框20 + 内边距40），右侧对称
		colB = ml
		colC = colB + 51    // 序号 0.75
		colD = colC + 271   // 货号及品名 4
		colE = colD + 68    // 单位 1
		colF = colE + 68    // 数量 1
		colG = colF + 85    // 单价 1.25
		colH = colG + 152   // 金额 2.25
		rightEdge = colH + 186 // 备注 2.75，= 941，右边距 = 980-941 = 39 ≈ 左边距
	)
	// 计算高度（动态商品行）
	const rowH = 50
	// 顶部区域高度：二维码(113) + 上边距(50) + 分隔线(20) + 客户信息两行(35+34+20)
	qrSizeCalc := 113
	sepYCalc := 50 + qrSizeCalc + 20
	var headerTopY int
	if isQuote {
		// 报价票据单：无客户信息，表格紧接标题分隔线
		headerTopY = sepYCalc + 35
	} else {
		infoEndYCalc := sepYCalc + 35 + 34 + 20
		headerTopY = infoEndYCalc + 35 // 与客户信息分隔线间距统一为35
	}
	itemCount := len(order.Items)
	tableBottomY := headerTopY + 40 + itemCount*rowH
	// 汇总区行数：报价票据仅1行(合计)，订单票据4行
	sumRowH := 50
	summaryH := sumRowH * 4
	if isQuote {
		summaryH = sumRowH // 仅合计行
	}
	// 说明区高度预估
	descLineCount := 1
	if ts.Description != "" {
		// 粗略估算：每行约 24 个中文字符
		runeCount := len([]rune(ts.Description))
		descLineCount = (runeCount / 24) + 1
		if descLineCount > 10 {
			descLineCount = 10
		}
	}
	// 底部：botB→说明(30间距)→说明行数*24→开单时间(10间距+15字高)→底部边距30
	descH := 30 + descLineCount*24 + 10 + 15 + 30
	H := tableBottomY + summaryH + descH

	d, err := newTicketDrawer(W, H)
	if err != nil {
		return nil, err
	}

	// ===== 整体外边框 =====
	d.rect(20, 20, W-20, H-20, d.border)

	// ===== ① 顶部标题区（行 4-5）=====
	qrSize := 113 // 原 170 的 2/3
	qrX := colB
	qrY := 50
	d.drawQRCode(ts.QRCodeURL, qrX, qrY, qrSize)
	if ts.QRCodeURL != "" {
		d.rect(qrX, qrY, qrX+qrSize, qrY+qrSize, d.border)
	}

	// 平台名称（Ma Shan Zheng 字体，加大）
	brandX := qrX + qrSize + 20
	// 联系方式底部与二维码底部平齐；平台名称与联系方式的间距=订单票据单与单号的间距(36px)
	qrBottom := qrY + qrSize
	contactY := qrBottom - 3 // faceSmall(15px)基线，使文本底部≈二维码底部
	brandY := contactY - 36  // 与订单票据单→单号的36px间距一致
	d.text(platformName, brandX, brandY, d.red, d.faceBrand)
	// 联系方式（灰色小字，平台名称下方，加"联系方式："前缀；联系方式2与1之间空4个字符）
	if ts.Contact != "" || ts.Contact2 != "" {
		contactText := "联系方式：" + ts.Contact
		if ts.Contact2 != "" {
			contactText += "    " + ts.Contact2 // 4个空格间距
		}
		d.text(contactText, brandX, contactY, d.gray, d.faceSmall)
	}

	// 右侧：票据标题 + 单号（+ 可选订单追踪二维码）
	trackQRSize := qrSize
	trackQRX := rightEdge - trackQRSize
	trackQRY := qrY
	var trackImg image.Image
	if !isQuote && ts.TrackQR && trackURL != "" {
		trackImg = generateTrackQRCode(trackURL)
		d.drawQRCodeImage(trackImg, trackQRX, trackQRY, trackQRSize)
		d.rect(trackQRX, trackQRY, trackQRX+trackQRSize, trackQRY+trackQRSize, d.border)
	}
	// 票据标题右对齐到二维码左侧（若有），否则右对齐到 rightEdge
	titleRightX := rightEdge
	if trackImg != nil {
		titleRightX = trackQRX - 15
	}
	// 标题文本：报价票据单 or 订单票据单
	titleText := "订单票据单"
	if isQuote {
		titleText = "报价票据单"
	}
	// 单号底部与追踪二维码底部平齐：二维码底部=qrY+qrSize，单号基线=qrBottom-3
	if isQuote {
		// 报价票据单：无单号，标题居中于二维码高度
		d.textRight(titleText, titleRightX, qrY+qrSize/2+13, d.black, d.faceTitle)
	} else {
		// 订单票据单：标题在上，单号在下，单号底部与二维码底部平齐
		titleY := qrBottom - 36 - 15 // 标题基线，使单号在下方且底部平齐
		d.textRight(titleText, titleRightX, titleY, d.black, d.faceTitle)
		d.textRight("单号："+order.Hash, titleRightX, qrBottom-3, d.gray, d.faceSmall)
	}

	// 标题区分隔线
	sepY := qrY + qrSize + 20
	d.hline(20, W-20, sepY, d.line)

	// ===== ② 客户信息行（行 6-7）=====
	if !isQuote {
		infoY := sepY + 35
		addr := strings.TrimSpace(order.Community + " " + order.Building + " " + order.UnitNo + " " + order.Room)
		d.text("客户地址："+addr, colB, infoY, d.black, d.face)
		d.textRight("联系人："+order.ContactName, rightEdge, infoY, d.black, d.face)
		d.text("客户备注："+order.Notes, colB, infoY+34, d.black, d.face)
		d.textRight("联系号码："+order.Phone, rightEdge, infoY+34, d.black, d.face)

		infoEndY := infoY + 34 + 20
		d.hline(20, W-20, infoEndY, d.line)
	}

	// ===== ③ 商品表头（行 8）=====
	hdrY := headerTopY
	hdrBottom := hdrY + 40
	// 表头底色
	d.fillRect(colB, hdrY, rightEdge, hdrBottom, d.light)
	headers := []struct {
		label string
		cx    int
	}{
		{"序号", (colB + colC) / 2},
		{"货号及品名", (colC + colD) / 2},
		{"单位", (colD + colE) / 2},
		{"数量", (colE + colF) / 2},
		{"单价", (colF + colG) / 2},
		{"金额", (colG + colH) / 2},
		{"备注", (colH + rightEdge) / 2},
	}
	for _, h := range headers {
		d.textCenter(h.label, h.cx, hdrY+27, d.black, d.faceBold)
	}
	// 表头边框
	d.cellBox(colB, hdrY, rightEdge, hdrBottom)
	// 列分隔线
	d.vline(colC, hdrY, hdrBottom, d.border)
	d.vline(colD, hdrY, hdrBottom, d.border)
	d.vline(colE, hdrY, hdrBottom, d.border)
	d.vline(colF, hdrY, hdrBottom, d.border)
	d.vline(colG, hdrY, hdrBottom, d.border)
	d.vline(colH, hdrY, hdrBottom, d.border)

	// ===== ④ 商品行（行 9+，动态）=====
	rowY := hdrBottom
	for i, item := range order.Items {
		rowBottom := rowY + rowH
		// 序号
		d.textCenter(fmt.Sprintf("%d", i+1), (colB+colC)/2, rowY+32, d.black, d.face)
		// 货号及品名
		d.text(item.ProductName, colC+8, rowY+32, d.black, d.face)
		// 单位
		d.textCenter(item.Unit, (colD+colE)/2, rowY+32, d.black, d.face)
		// 数量
		d.textCenter(fmt.Sprintf("%d", item.Qty), (colE+colF)/2, rowY+32, d.black, d.face)
		// 单价
		d.textCenter(toYuan(item.PriceCents), (colF+colG)/2, rowY+32, d.black, d.face)
		// 金额（退货取负）
		amt := item.LineCents
		remark := ""
		if item.IsReturn {
			amt = -amt
			remark = "退货"
		}
		d.textCenter(toYuan(amt), (colG+colH)/2, rowY+32, d.black, d.face)
		// 备注
		if remark != "" {
			d.textCenter(remark, (colH+rightEdge)/2, rowY+32, d.red, d.face)
		}
		// 行边框
		d.cellBox(colB, rowY, rightEdge, rowBottom)
		d.vline(colC, rowY, rowBottom, d.border)
		d.vline(colD, rowY, rowBottom, d.border)
		d.vline(colE, rowY, rowBottom, d.border)
		d.vline(colF, rowY, rowBottom, d.border)
		d.vline(colG, rowY, rowBottom, d.border)
		d.vline(colH, rowY, rowBottom, d.border)
		rowY = rowBottom
	}

	// ===== ⑤ 汇总区 =====
	s1Y := rowY
	s1B := s1Y + sumRowH

	// 行25: 合计行（报价与订单都有）
	d.text("总计金额（大写）："+chineseCapital(order.TotalCents), colB+10, s1Y+32, d.black, d.faceBold)
	d.cellBox(colB, s1Y, colF, s1B) // B25:E25
	d.textCenter("合计", (colF+colG)/2, s1Y+32, d.black, d.face)
	d.cellBox(colF, s1Y, colG, s1B)
	d.textCenter(toYuan(order.TotalCents), (colG+rightEdge)/2, s1Y+32, d.black, d.faceBold)
	d.cellBox(colG, s1Y, rightEdge, s1B)

	var botB int
	if isQuote {
		// 报价票据单：仅合计行，无后续汇总
		botB = s1B
	} else {
		// 订单票据单：完整汇总区
		s2Y := s1B
		s2B := s2Y + sumRowH
		s3Y := s2B
		s3B := s3Y + sumRowH

		// 行26: 抹零/优惠 + 应付
		due := order.TotalCents - order.DiscountCents
		d.text("抹零/优惠金额："+toYuan(order.DiscountCents), colB+10, s2Y+32, d.black, d.face)
		d.cellBox(colB, s2Y, colF, s2B)
		d.textCenter("应付", (colF+colG)/2, s2Y+32, d.black, d.face)
		d.cellBox(colF, s2Y, colG, s2B)
		d.textCenter(toYuan(due), (colG+rightEdge)/2, s2Y+32, d.black, d.faceBold)
		d.cellBox(colG, s2Y, rightEdge, s2B)

		// 行27+28: 备注(跨2行) + 待付(1行) + 结账方式/确认
		unpaid := due - order.PaidCents
		botY := s3B
		botB = botY + sumRowH
		d.text("备注："+order.Notes, colB+10, s3Y+32, d.black, d.face)
		d.cellBox(colB, s3Y, colF, botB)
		d.textCenter("待付", (colF+colG)/2, s3Y+32, d.black, d.face)
		d.cellBox(colF, s3Y, colG, s3B)
		d.textCenter(toYuan(unpaid), (colG+rightEdge)/2, s3Y+32, d.red, d.faceBold)
		d.cellBox(colG, s3Y, rightEdge, s3B)

		// 行28: 结账方式 / 确认
		d.text("结账方式：", colF+10, botY+32, d.black, d.face)
		d.cellBox(colF, botY, colH, botB)
		d.text("确认：", colH+10, botY+32, d.black, d.face)
		d.cellBox(colH, botY, rightEdge, botB)
	}

	// 行29：说明（换行）
	descY := botB + 30
	var descLines []string
	if ts.Description != "" {
		descLines = d.wrapText("说明："+ts.Description, rightEdge-colB-20, d.faceSmall)
		for i, line := range descLines {
			d.text(line, colB+10, descY+i*24, d.gray, d.faceSmall)
		}
	}
	// 开单时间靠右，放在说明文本下方避免重叠
	timeY := descY
	if len(descLines) > 0 {
		timeY = descY + len(descLines)*24 + 10
	}
	d.textRight("开单时间："+order.CreatedAt, rightEdge-10, timeY, d.gray, d.faceSmall)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, d.img, &jpeg.Options{Quality: 92}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
