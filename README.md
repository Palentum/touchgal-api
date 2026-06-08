# TouchGal Developer/API

TouchGal Developer/API 是一个完全独立于 KunMoe/kun-touchgal-next 主项目的开发者 API 项目，包含 Go 后端、Nuxt 前端、独立 PostgreSQL、Redis、同步 Worker、OpenAPI 文档和 Docker Compose 部署示例。

## 架构图文字版

```
KunMoe 主项目 PostgreSQL（只读账号）
  -> backend sync worker（full / incremental）
  -> TouchGal Developer 独立 PostgreSQL clean DB
  -> Go /v1 API（API token + Redis 限流）
  -> Nuxt Developer Portal（HttpOnly session cookie）
```

## 数据流说明

主项目数据库 -> 同步任务 -> 独立净化库 -> API。

同步任务只读取 `patch`、`patch_alias`、`patch_tag_relation + patch_tag`、`patch_company_relation + patch_company`、`patch_rating_stat` 的条目元数据和评分聚合。写入本项目 `games`、`game_aliases`、`tags`、`companies`、`game_rating_stats` 等 clean tables。

## 为什么不直接读主库对外提供 API

- 主库包含用户、评论、资源链接、权限、会话等敏感业务数据。
- 公开 API 需要独立限流、申请审核、审计和可撤销 token。
- clean DB 可以固定公开契约，避免主站内部 schema 变化直接影响开发者。
- 同步层可以统一脱敏、下架校准和搜索索引生成。

## 脱敏策略

永不同步：主项目 user 表、用户 email/password/IP/role/session/token、评论、私信、举报、收藏、文件夹、资源站下载链接、上传者 user_id、主库内部自增 id。

公开 API 永不返回：`source_patch_id`、主站用户信息、资源站链接、评论、上传者、主库内部 id。第一版默认只公开 `content_limit = 'sfw'` 且 `deleted_at is null` 的条目。

## 环境变量说明

后端复制：

```bash
cp backend/.env.example backend/.env
```

关键变量：

- `DATABASE_DSN`：本项目独立 PostgreSQL。
- `SOURCE_DATABASE_DSN`：KunMoe 主库只读账号连接串，生产建议 `sslmode=require`。
- `REDIS_ADDR` / `REDIS_PASSWORD` / `REDIS_DB`：验证码、session cache、API token 限流。
- `SESSION_SECRET`：登录 session hash secret。
- `SMTP_*`：邮箱验证码 SMTP。
- `API_TOKEN_PEPPER`：API token hash pepper，数据库只存 `sha256(token + "." + pepper)`。
- `ENABLE_SYNC_WORKER`：API 进程是否启动后台同步。
- `SYNC_INTERVAL_MINUTES` / `SYNC_FULL_INTERVAL_HOURS`：incremental/full 同步周期。

前端复制：

```bash
cp frontend/.env.example frontend/.env
```

## 本地启动

```bash
make dev
```

或分别启动：

```bash
make backend-dev
make frontend-dev
```

## 数据库迁移

安装 goose 后执行：

```bash
export DATABASE_DSN='postgres://touchgal_api:touchgal_api@localhost:5432/touchgal_api?sslmode=disable'
make migrate-up
```

## 执行同步

```bash
make sync       # go run ./cmd/sync --mode=incremental
make sync-full  # go run ./cmd/sync --mode=full
```

`SOURCE_DATABASE_DSN` 必须使用主库只读账号。本项目禁止修改主项目数据库。

## 创建管理员方式

先通过前端注册账号，然后在本项目独立库中执行：

```sql
UPDATE users SET is_admin = true WHERE email = 'admin@example.com';
```

不要复制或复用主站账号权限。

## 邮箱 SMTP 配置

配置 `SMTP_HOST`、`SMTP_PORT`、`SMTP_USERNAME`、`SMTP_PASSWORD`、`SMTP_FROM`、`SMTP_FROM_NAME`。开发环境未配置 SMTP 时，后端会将验证码写入结构化日志；生产必须配置真实 SMTP。

## API token 申请流程

1. 用户通过邮箱验证码注册或登录。
2. 登录后访问 `/apply` 提交申请：负责人、项目地址、预估请求量、使用场景、同意条款。
3. 管理员在 `/admin/applications` 审核。
4. 申请 `approved` 后，用户在 `/dashboard/tokens` 创建 token。
5. token 明文只返回一次；数据库只保存 hash。
6. 用户可随时失效 token；revoked token 不能继续访问 `/v1`。

## API 调用示例

搜索：

```bash
curl "https://api.example.com/v1/games/search?keyword=summer&page=1&limit=10" \
  -H "Authorization: Bearer tgal_live_xxx"
```

详情：

```bash
curl "https://api.example.com/v1/games/abcd1234" \
  -H "Authorization: Bearer tgal_live_xxx"
```

Token 自检：

```bash
curl "https://api.example.com/v1/me" \
  -H "Authorization: Bearer tgal_live_xxx"
```

错误响应：

```json
{
  "success": false,
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Missing or invalid API token"
  }
}
```

## 部署建议

- 使用 `deploy/docker-compose.yml` 本地或小规模部署。
- 生产使用独立 PostgreSQL、Redis、只读 SOURCE DB 账号。
- 可将 API HTTP 与 sync worker 分离：API 设置 `ENABLE_SYNC_WORKER=false`，使用 k8s CronJob 或系统 cron 执行 `touchgal-sync --mode=incremental`。
- 推荐 nginx/Ingress 终止 TLS，并将 `SESSION_COOKIE_SECURE=true`。

## 安全注意事项

- 登录 session 只通过 HttpOnly Cookie 保存，前端不写 localStorage。
- API token 明文只显示一次，日志不得记录明文 token。
- 邮箱验证码只存 hash，并有 TTL、冷却和最大尝试次数。
- 管理接口必须 `users.is_admin=true`。
- 公开 API 默认只返回 SFW 条目。
- 响应错误不暴露数据库结构。
- clean DB 不包含主项目 user/email/password/IP/role/session/token/resource link。
