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
docker build -t zhiqiangwang/app:llm-gate .
docker run --rm -p 8080:80 zhiqiangwang/app:llm-gate
```

### 默认账号

- 邮箱: `admin@example.com`
- 密码: `123456`

## 配置

| 字段 | 说明 | 默认值 |
|---|---|---|
| `server.host` / `server.port` | 监听地址 / 端口 | `0.0.0.0` / `8080` |
| `db.driver` / `db.database` | 数据库驱动 / 文件或名称 | `sqlite` / `./data.db` |
| `jwt.secret` | JWT 密钥 | `llm-gate-secret-key` |
| `jwt.access_token_expire` | 访问令牌过期 | `2h` |
| `jwt.refresh_token_expire` | 刷新令牌过期 | `168h` |
| `relay.timeout` | 转发请求超时（秒） | `120` |
| `relay.max_body_mb` | 最大请求体（MB） | `32` |
| `relay.pre_consume_cents` | 请求预扣金额（分） | `100` |
| `relay.stream_fallback_cents` | 流式无 usage 时兜底扣费（分） | `100` |
| `relay.rate_limit.*` | Token/账户/全局限流（enabled/rate/burst/global_rate/account_rate） | 关 |
| `relay.upstream.*` | 非流式失败重试（retry_enabled/max_retries/retry_delay_ms） | 开/2/200 |
| `relay.failover.*` | 多服务商降级熔断（threshold/window/cooldown/health_probe） | 开/5/60s/30s |
| `billing.base_price_cents_per_1k` | 每千 token 基础单价（分） | `2` |
| `billing.min_balance_cents` | 余额低于该值触发告警（分） | `1000` |
| `security.encrypt_key` | AES-256 加密密钥（64 位 hex），服务商/Token 密钥加密存储 | 空（回退 JWT Secret） |
| `security.reveal_audit` | 记录密钥查看审计日志 | `true` |
| `retention.usage_days` / `transaction_days` | 用量日志 / 资金流水保留天数，≤0 不清理 | `0` |
| `retention.expired_token_grace_days` | 过期 Token 宽限期清理，≤0 不清理 | `0` |
| `alert.*` | Webhook 告警（enabled/webhook_url/sign_secret/cooldown） | 关 |
| `redis.addr` | 缓存 Redis 地址，空则用内存缓存 | 空 |

支持环境变量覆盖：`JWT_SECRET`、`ENCRYPT_KEY`。

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
| GET/POST/PUT/DELETE | `/llm/providers[/:id]` | 服务商 CRUD（密钥加密存储） |
| GET | `/llm/providers/all` | 所有服务商 |
| GET | `/llm/providers/:id/sync-models/preview` | 预览上游模型 |
| POST | `/llm/providers/:id/sync-models` | 同步上游模型 |
| GET/POST/PUT/DELETE | `/llm/models[/:id]` | 模型 CRUD |
| GET | `/llm/models/all` | 所有模型 |
| GET/POST/PUT/DELETE | `/llm/tokens[/:id]` | API Key CRUD（密钥加密存储） |
| GET | `/llm/tokens/:id/reveal` | 查看完整密钥（需所有权，记录审计日志） |
| GET | `/llm/usage` | 用量日志 |
| GET | `/llm/usage/stats` | 用量统计 |
| GET | `/billing/orders` | 充值订单列表 |
| POST | `/billing/orders` | 创建充值订单 |
| POST | `/billing/orders/:id/confirm` | 确认充值入账 |
| POST | `/billing/orders/:id/cancel` | 取消充值订单 |
| GET | `/billing/transactions` | 账户资金流水 |
| POST | `/billing/balance/adjust` | 手动调整余额 |

### 转发 API（Bearer Token 认证）

前缀 `/v1`，需在 Header 中携带 `Authorization: Bearer sk-<hex>`。

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/v1/chat/completions` | 对话补全（流式/非流式） |
| POST | `/v1/embeddings` | 向量嵌入 |
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
| key | string | 密钥，AES-256 加密存储 |
| key_hash | string | 密钥 SHA256，用于认证查询（明文不可恢复） |
| quota | int64 | 预算（分），0=不限，超出后拒绝 |
| spent_cents | int64 | 已消费（分） |
| model_ids | string | 模型白名单 JSON 数组，空=全部 |
| expired_at | *time.Time | 过期时间 |
| status | bool | 启用/禁用 |

密钥生成: 16 随机字节 → `sk-` + hex 编码（35 字符），库内仅存密文与哈希。

### 充值订单 (llm_recharge_orders)

真实货币计费：线下转账后由管理员确认入账。

| 字段 | 类型 | 说明 |
|---|---|---|
| account_id | int64 | 账户 ID |
| amount_cents | int64 | 充值金额（分） |
| status | string | pending / paid / cancelled |
| created_by / paid_by | int64 | 创建人 / 确认人 |
| paid_at | *time.Time | 确认时间 |

### 资金流水 (llm_transactions)

| 字段 | 类型 | 说明 |
|---|---|---|
| account_id | int64 | 账户 ID |
| type | string | consume / refund / recharge / adjust |
| amount_cents | int64 | 变动金额（分，可正可负） |
| balance_cents | int64 | 变动后余额（分） |
| token_id / request_id | - | 关联 Token / 请求 |

### 用量日志 (llm_usage_logs)

| 字段 | 类型 | 说明 |
|---|---|---|
| account_id | int64 | 账户 ID |
| token_id | int64 | 令牌 ID |
| model_name | string | 模型名称 |
| provider_id | int64 | 服务商 ID |
| prompt_tokens / completion_tokens / total_tokens | int | 令牌数 |
| cost_cents | int64 | 实际扣费（分） |
| estimated | bool | 是否估算计费（流式未返回 usage） |
| request_id | string | 请求 ID |

## 模型分发

同名模型配置多个服务商时，使用**加权随机**分发：

1. 从缓存（30s）或 DB 查询所有同名的启用模型
2. 过滤 Token 模型白名单
3. 按权重选取主服务商，其余作为候选按权重排序
4. 将请求中 `model` 字段替换为 `upstream_model_name`
5. 主服务商失败时自动切换候选（熔断降级）

示例：`deepseek-chat` 配了 DeepSeek (weight=2) 和 阿里云 (weight=1)，约 2/3 请求走 DeepSeek，1/3 走阿里云。

## 计费与配额

真实货币计费，单位分（1 元 = 100 分）：

```txt
单次费用 = (prompt/1000×ratio + completion/1000×ratio×completion_ratio) × base_price_cents_per_1k
```

预扣 + 结算模式：

1. 请求开始时预扣 `pre_consume_cents`（默认 100 分）
2. 非流式响应结束读取 `usage` 计算实际费用，多退少补并写资金流水
3. 流式请求在流结束后解析 usage 结算；未返回 usage 时按 `stream_fallback_cents` 兜底并标记 `estimated`
4. Token 预算（quota）在认证时校验，超出拒绝
5. 余额不足（< 预扣）时拒绝请求并告警
6. 结算通过异步队列批量处理（channel 1024 / 4 worker），失败退回首轮预扣
7. 用量日志通过批量写入（最大 32 条 / 3s 间隔）

## 可靠性

- **熔断降级**：同模型多服务商时，窗口内连续失败达到阈值自动熔断 30s，跳过熔断服务商并告警
- **健康探测**：周期探测上游 `/models` 恢复熔断状态
- **流式排水**：客户端断连后继续读取上游响应以完成 usage 计费
- **重试**：仅非流式重试（SSE 不可重放），4xx 直接透传
- **数据保留**：按 `retention.*` 策略周期清理用量日志/资金流水/过期 Token
- **Webhook 告警**：余额不足、熔断跳过时推送，HMAC-SHA256 签名 + 冷却去重

## 缓存策略

| 缓存 | 过期 | 内容 |
|---|---|---|
| authCache | 10s | Token 认证结果 |
| modelListCache | 30s | 模型配置列表 `[]ModelConfig` |
| providerCache | 60s | 服务商配置 |

## 项目结构

```txt
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
    ├── relay.go              # 入口：认证、模型解析、转发、计费
    ├── forward.go            # HTTP 转发 + 流式代理 + 降级
    ├── auth.go               # Token 认证
    ├── resolver.go           # 模型解析
    ├── batch.go              # 用量批量写入
    ├── breaker.go            # 熔断器
    ├── notify.go             # Webhook 告警
    └── usage.go              # 计费原语
├── security/
│   └── crypto.go             # AES-GCM 加解密 + SHA256
└── web/                      # Next.js 前端
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
- **完整菜单树**: 仪表盘、系统管理（账号/角色/菜单/日志）、LLM 网关（服务商/模型/API Key）、计费中心（充值订单/资金流水）

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
