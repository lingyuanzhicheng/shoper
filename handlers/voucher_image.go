package handlers

import (
	"net/http"
	"path/filepath"
	"strings"

	"shoper/db"
	"shoper/middleware"
	"shoper/models"
)

// voucherImageHandler 服务订单凭证图片，禁止匿名直链访问。
// 鉴权策略：
//   - 管理员（admin cookie）：直接放行
//   - 其他用户：必须携带 hash + phone 查询参数，校验订单存在、手机号匹配、
//     状态非「已取消」「已归档」、文件名属于该订单的凭证列表
func voucherImageHandler(w http.ResponseWriter, r *http.Request) {
	// 从路径提取文件名（如 /uploads/voucher/xxx.jpg → xxx.jpg）
	filename := strings.TrimPrefix(r.URL.Path, "/uploads/voucher/")
	filename = filepath.Base(filename)
	if filename == "" || filename == "." || filename == "/" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// 管理员直接放行
	if middleware.IsAdmin(r) {
		http.ServeFile(w, r, filepath.Join("uploads", "voucher", filename))
		return
	}

	// 非管理员：必须有 hash + phone 查询参数
	hash := strings.TrimSpace(r.URL.Query().Get("hash"))
	phone := strings.TrimSpace(r.URL.Query().Get("phone"))
	if hash == "" || phone == "" {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// 校验订单
	order, err := db.GetOrderByHashForCustomer(hash)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if order.Phone != phone {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	// 已取消或已归档禁止访问
	if order.Status == models.StatusCancelled || order.Status == models.StatusArchived || order.Archived {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// 校验文件名属于该订单的凭证
	vouchers, err := db.GetOrderVouchers(hash)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	allowed := false
	for _, v := range vouchers {
		if filepath.Base(v.ImageURL) == filename {
			allowed = true
			break
		}
	}
	if !allowed {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	http.ServeFile(w, r, filepath.Join("uploads", "voucher", filename))
}
