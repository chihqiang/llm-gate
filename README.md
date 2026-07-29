# llm-gate

聚合多 LLM 服务商，提供统一 OpenAI 兼容接口的 API 网关。内置管理后台，支持服务商管理、模型管理、API Key 管理、用量统计。

## 功能

- **统一接口**：对外提供 OpenAI 兼容的 `/v1/chat/completions` 和 `/v1/models` 接口
- **多服务商聚合**：可配置多个上游服务商（如 DeepSeek、阿里云等），同一模型可跨服务商部署
- **API Key 管理**：支持多用户、多 Key，配额控制，过期时间
- **用量统计**：记录每次请求的 Token 消耗和配额扣减，提供按模型的聚合统计
- **自动同步模型**：从上游服务商拉取模型列表，选择后一键导入
- **管理后台**：Next.js 构建的现代化管理界面，RBAC 权限控制

## 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go, [infra-go](https://github.com/chihqiang/infra-go)（HTTP 框架）, GORM, SQLite/MySQL/PostgreSQL |
| 前端 | Next.js 16, React 19, shadcn/ui, Tailwind CSS v4, Recharts |
| 认证 | JWT（管理端）, Bearer Token / sk-xxx（API 端） |

## 快速开始

### 1. 启动后端

```bash
go run main.go
```

后端默认监听 `:8080`，使用 SQLite 存储，首次启动自动建表和填充种子数据。

### 2. 启动前端

```bash
cd web
pnpm install
pnpm dev
```

前端默认监听 `:3000`，访问 `http://localhost:3000` 进入管理后台。

默认管理员账号：`admin@example.com` / `123456`

### 3. 配置上游服务商

进入管理后台 → **LLM 网关** → **服务商管理**，添加服务商（如 DeepSeek、阿里云等），然后点击「同步模型」拉取上游模型列表，选择需要的模型导入。

### 4. 创建 API Key

进入 **LLM 网关** → **API Key** 创建密钥，调用 API 时使用 `Authorization: Bearer sk-xxx`。

### 5. 调用 API

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer sk-xxxxx" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-flash",
    "messages": [{"role": "user", "content": "hello"}],
    "stream": false
  }'
```

## 配置

编辑 `config.yaml`：

```yaml
server:
  host: 0.0.0.0
  port: 8080

db:
  driver: sqlite        # sqlite | mysql | postgres
  database: ./data.db

relay:
  timeout: 120          # 上游请求超时（秒）
  max_body_mb: 32       # 最大请求体（MB）

jwt:
  secret: llm-gate-secret-key
  access_token_expire: 2h
  refresh_token_expire: 168h
```

## 项目结构

```
├── main.go                 # 入口
├── config.yaml             # 配置
├── config/config.go        # 配置结构体
├── db/migrate.go           # 数据库迁移 + 种子数据
├── model/                  # GORM 模型
│   ├── provider.go         # 服务商
│   ├── model_config.go     # 模型配置
│   ├── user_token.go       # API Key
│   └── usage_log.go        # 用量日志
├── logic/                  # 业务逻辑
│   ├── provider.go         # 服务商管理、同步模型
│   ├── model_config.go     # 模型管理
│   ├── user_token.go       # Key 管理、配额扣减
│   ├── usage.go            # 用量记录与统计
│   └── dashboard.go        # 仪表盘统计
├── handler/                # HTTP Handler
├── middleware/             # 认证、权限中间件
├── relay/                  # 转发引擎
│   ├── auth.go             # API Key 认证
│   ├── resolver.go         # 模型→服务商解析
│   ├── forward.go          # 上游请求转发
│   └── relay.go            # Chat/Models Handler
├── route/route.go          # 路由注册
└── web/                    # Next.js 前端
    ├── api/                # API 调用层
    ├── components/         # UI 组件
    └── app/admin/          # 管理后台页面
        ├── dashboard/      # 数据概览
        └── sys/llm/        # LLM 网关管理
            ├── providers/  # 服务商管理
            ├── models/     # 模型管理
            └── tokens/     # API Key
```

## API 概览

### 管理端 API（需 JWT 认证）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/api/v1/auth/login` | 登录 |
| GET | `/api/v1/dashboard/stats` | 仪表盘统计 |
| CRUD | `/api/v1/llm/providers` | 服务商管理 |
| CRUD | `/api/v1/llm/models` | 模型管理 |
| CRUD | `/api/v1/llm/tokens` | API Key 管理 |
| GET | `/api/v1/llm/tokens/{id}/reveal` | 获取完整 Key（仅本人） |
| POST | `/api/v1/llm/providers/{id}/sync-models` | 同步上游模型 |
| GET | `/api/v1/llm/providers/{id}/sync-models/preview` | 预览上游模型 |
| GET | `/api/v1/llm/usage` | 用量明细 |
| GET | `/api/v1/llm/usage/stats` | 用量聚合统计 |

### 转发 API（需 Bearer Token / sk-xxx）

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/v1/chat/completions` | Chat Completion（支持 streaming） |
| GET | `/v1/models` | 可用模型列表 |
