# Shoper

Shoper 是一个轻量级的商品目录与订单管理平台，适用于家居产品的展示和销售。项目使用 Go 编写，基于标准库 `net/http` 提供 HTTP 服务，SQLite 作为数据存储，部署简单、开箱即用。

## 功能特性

- **商品管理**：支持商品分类、品牌、多图展示，可按分类/品牌/关键词筛选
- **订单流程**：购物车 → 提交订单 → 生成订单号 → 凭订单号和联系电话追踪状态
- **管理后台**：商品、品牌、分类、订单、媒体文件的完整 CRUD 管理
- **工单/票据**：自动生成订单票据图片，支持二维码
- **页面配置**：首页轮播、关于页、平台名称等均可在后台自定义
- **嵌入式资源**：模板、静态文件、字体通过 `embed.FS` 打包，单一二进制部署

## 技术栈

- **语言**：Go 1.22+
- **数据库**：SQLite (go-sqlite3)
- **前端**：Alpine.js + Tailwind CSS（无构建步骤，直接引用静态文件）
- **字体**：Noto Sans SC / Ma Shan Zheng（用于票据渲染）

## 快速开始

### 本地运行

```bash
go build -o shoper .
./shoper
```

默认监听 `http://localhost:8080`，默认管理员账号 `shoper / shoper`。

### Docker 部署

```bash
docker compose up -d
```

服务启动后访问 `http://localhost:8080`。

## 配置

通过环境变量配置管理员账号：

| 环境变量 | 说明 | 默认值 |
|---------|------|-------|
| `SHOPER_ADMIN_USERNAME` | 管理员用户名 | `shoper` |
| `SHOPER_ADMIN_PASSWORD` | 管理员密码 | `shoper` |

## 目录结构

```
shoper/
├── main.go              # 程序入口
├── db/                  # 数据库操作层
├── handlers/            # HTTP 请求处理器
├── middleware/          # 中间件（认证等）
├── models/              # 数据模型
├── utils/               # 工具函数
├── templates/           # HTML 模板
├── static/              # 静态资源（CSS/JS/字体）
├── assets/              # 嵌入式资源（字体/favicon）
├── data/                # SQLite 数据库（运行时生成）
└── uploads/             # 上传文件（运行时生成）
```

## 许可证

详见 [LICENSE](LICENSE)。
