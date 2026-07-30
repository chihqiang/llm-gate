# llm-gate

API 网关，聚合多个 LLM 服务商，对外提供统一 OpenAI 兼容接口。

## 架构

```bash
客户端 (OpenAI SDK) → Nginx (80) → /api/* → Go :8080 (管理端)
                                  → /*    → Next.js :3000 (前端)
```

- **Go 后端**: HTTP 服务 (:8080)，JWT 管理 API + Bearer Token 转发 API
- **Next.js 前端**: 管理面板，React 19 + shadcn/ui，部署为 standalone server
- **Nginx**: 反向代理，统一入口 80 端口
- **Supervisord**: 容器内管理三个进程（gate / nextjs / nginx）

## 技术栈

| 层 | 技术 |
|---|---|
| 后端语言 | Go 1.25 |
| HTTP 框架 | `github.com/chihqiang/infra-go/httpx` |
| ORM | GORM v1.31，支持 SQLite / MySQL / PostgreSQL |
| 认证 | JWT (管理端) / Bearer Token `sk-*` (转发端) |
| 缓存 | `github.com/patrickmn/go-cache`（三级缓存） |
| 日志 | zap 结构化日志 |
| 前端框架 | Next.js 16 + React 19 + TypeScript |
| UI 组件 | shadcn/ui + Tailwind CSS v4 |
| 图表 | Recharts |
| HTTP 客户端 | Axios |

## 快速开始

### 本地开发

**后端**:

```bash
go run main.go
```

后端监听 `:8080`，使用 SQLite 数据库。

**前端**:

```bash
cd web
pnpm dev
```

前端监听 `:3000`，自动代理 `/api/*` 到后端。

### Docker 部署

```bash
docker build -t llm-gate .
docker run --rm -p 8080:80 llm-gate
```

### 默认账号

- 邮箱: `admin@example.com`
- 密码: `123456`

## 配置

| 字段 | 说明 | 默认值 |
|---|---|---|
| `server.host` | 监听地址 | `0.0.0.0` |
| `server.port` | 监听端口 | `8080` |
| `db.driver` | 数据库驱动 | `sqlite` |
| `db.database` | 数据库文件/名称 | `./data.db` |
| `jwt.secret` | JWT 密钥 | `llm-gate-secret-key` |
| `jwt.access_token_expire` | 访问令牌过期 | `2h` |
| `jwt.refresh_token_expire` | 刷新令牌过期 | `168h` |
| `relay.timeout` | 转发请求超时 | `120` |
| `relay.max_body_mb` | 最大请求体 | `32` |

## API

### 管理 API（JWT 认证）

所有管理接口前缀 `/api/v1`，需在 Header 中携带 `Authorization: Bearer <jwt_token>`。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/auth/login` | 登录 |
| GET | `/auth/me` | 当前用户信息 |
| GET/POST/PUT/DELETE | `/sys/accounts[/:id]` | 账号 CRUD |
| GET/POST/PUT/DELETE | `/sys/roles[/:id]` | 角色 CRUD |
| POST | `/sys/roles/:id/menus` | 角色关联菜单 |
| GET/POST/PUT/DELETE | `/sys/menus[/:id]` | 菜单 CRUD |
| GET | `/sys/logs` | 操作日志 |
| GET | `/dashboard/stats` | 仪表盘统计 |
| GET/POST/PUT/DELETE | `/llm/providers[/:id]` | 服务商 CRUD |
| GET | `/llm/providers/all` | 所有服务商 |
| GET | `/llm/providers/:id/sync-models/preview` | 预览上游模型 |
| POST | `/llm/providers/:id/sync-models` | 同步上游模型 |
| GET/POST/PUT/DELETE | `/llm/models[/:id]` | 模型 CRUD |
| GET | `/llm/models/all` | 所有模型 |
| GET/POST/PUT/DELETE | `/llm/tokens[/:id]` | API Key CRUD |
| GET | `/llm/tokens/:id/reveal` | 查看完整密钥（需所有权） |
| GET | `/llm/usage` | 用量日志 |
| GET | `/llm/usage/stats` | 用量统计 |

### 转发 API（Bearer Token 认证）

前缀 `/v1`，需在 Header 中携带 `Authorization: Bearer sk-<hex>`。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/v1/chat/completions` | 对话补全（流式/非流式） |
| GET | `/v1/models` | 可用模型列表 |

## 数据模型

### 服务商 (llm_providers)

| 字段 | 类型 | 说明 |
|---|---|---|
| name | string | 名称，如 DeepSeek |
| base_url | string | API 地址 |
| api_key | string | API 密钥 |
| status | bool | 启用/禁用 |
| priority | int | 优先级 |
| weight | int | 权重 |

### 模型 (llm_models)

| 字段 | 类型 | 说明 |
|---|---|---|
| name | string | 对外模型名，如 deepseek-chat |
| provider_id | int64 | 关联服务商 |
| upstream_model_name | string | 上游实际模型名 |
| model_ratio | float64 | 模型倍率 |
| completion_ratio | float64 | 补全倍率 |
| weight | int | 权重（同模型多服务商时按权重分发） |
| status | bool | 启用/禁用 |

`(provider_id, upstream_model_name)` 联合唯一索引。

### API Key (llm_user_tokens)

| 字段 | 类型 | 说明 |
|---|---|---|
| account_id | int64 | 所属账户 |
| name | string | 令牌名称 |
| key | string | 密钥，`sk-` + 32 位 hex |
| quota | int64 | 额度 |
| expired_at | *time.Time | 过期时间 |
| status | bool | 启用/禁用 |

密钥生成: 16 随机字节 → `sk-` + hex 编码（35 字符）。

### 用量日志 (llm_usage_logs)

| 字段 | 类型 | 说明 |
|---|---|---|
| account_id | int64 | 账户 ID |
| token_id | int64 | 令牌 ID |
| model_name | string | 模型名称 |
| provider_id | int64 | 服务商 ID |
| prompt_tokens | int | 提示令牌 |
| completion_tokens | int | 补全令牌 |
| total_tokens | int | 总令牌 |
| quota_cost | int64 | 额度消耗 |
| request_id | string | 请求 ID |

## 模型分发

同名模型配置多个服务商时，使用**加权随机**策略分发：

1. 从缓存（30s）或 DB 查询所有同名的启用模型
2. 计算总权重
3. 按 `time.Now().UnixNano() % totalWeight` 选取
4. 将请求中 `model` 字段替换为 `upstream_model_name`

示例：`deepseek-chat` 配了 DeepSeek (weight=2) 和 阿里云 (weight=1)，约 2/3 请求走 DeepSeek，1/3 走阿里云。

## 缓存策略

| 缓存 | 过期 | 内容 |
|---|---|---|
| authCache | 10s | Token 认证结果 |
| modelListCache | 30s | 模型配置列表 `[]ModelConfig` |
| providerCache | 60s | 服务商配置 |

## 配额

预扣 + 结算模式：

1. 请求开始时预扣 1000 额度
2. 非流式响应结束后读取 `usage` 字段计算实际消耗
3. 多退少补（`delta = actual - preConsume`）
4. 流式请求未计入用量（配额退还）
5. 用量日志通过批量写入（最大 32 条 / 3s 间隔）

## 项目结构

```
├── main.go                   # 入口
├── config.yaml               # 配置文件
├── Dockerfile                # 多阶段构建
├── nginx.conf                # Nginx 配置
├── supervisord.conf          # Supervisor 配置
├── config/
│   └── config.go             # 配置结构体
├── db/
│   └── migrate.go            # 数据库迁移 + 种子数据
├── model/                    # 数据模型
├── logic/                    # 业务逻辑层
├── handler/                  # HTTP 处理器
├── middleware/                # 中间件（认证/日志/权限）
├── route/
│   └── route.go              # 路由注册
└── relay/                    # LLM 转发核心
    ├── relay.go              # 入口：认证、模型解析、转发、配额
    ├── forward.go            # HTTP 转发 + 流式代理
    ├── auth.go               # Token 认证
    ├── resolver.go           # 模型解析
    ├── batch.go              # 用量批量写入
    └── usage.go              # 配额扣减
├── web/                      # Next.js 前端
    ├── api/                  # API 请求封装
    ├── app/                  # 页面路由
    ├── components/           # UI 组件
    ├── hooks/                # React Hooks
    └── lib/                  # 工具函数
```

## 种子数据

首次启动自动创建：

- **超级管理员** 角色，关联所有菜单权限
- **管理员账号** admin@example.com / 123456
- **完整菜单树**: 仪表盘、系统管理（账号/角色/菜单/日志）、LLM 网关（服务商/模型/API Key）

## Docker 构建

多阶段构建：

| 阶段 | 基础镜像 | 产物 |
|---|---|---|
| go-builder | golang:1.25-alpine | Go 二进制（CGO 启用） |
| node-builder | node:22-alpine | Next.js standalone 构建产物 |
| runtime | node:22-alpine | 聚合所有产物 + nginx + supervisord |

镜像包含 3 个进程（supervisord 管理）：
- `gate` — Go 后端
- `nextjs` — Next.js 服务（Node）
- `nginx` — 反向代理

所有进程日志输出到 stdout/stderr。
