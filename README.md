# TouchGal API

TouchGal API 是一个完全独立于 TouchGal 主项目（`kun-touchgal-next`）的开发者 API 项目，包含 Go 后端、Nuxt 前端、同步 Worker、OpenAPI 文档和 Docker Compose 部署示例；PostgreSQL 与 Redis 由主机提供，不再由项目编排启动。

## 架构图文字版

```
TouchGal 主项目 PostgreSQL（只读账号）
  -> backend sync worker（full / incremental）
  -> TouchGal API 独立 PostgreSQL clean DB
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

- `DATABASE_DSN`：本项目 clean DB，指向主机上的 PostgreSQL。
- `SOURCE_DATABASE_DSN`：TouchGal 主库只读账号连接串，生产建议 `sslmode=require`；默认只配置给独立 sync worker，API 进程在 `ENABLE_SYNC_WORKER=false` 时不需要该凭据。
- `DB_*` / `SYNC_DB_*` / `SOURCE_DB_*`：分别控制 API clean DB、sync clean DB、source 主库连接池与 `statement_timeout`、`idle_in_transaction_session_timeout`、query timeout；sync 使用独立 target/source pool、分页读取和短事务 batch commit，避免 full sync 占用 API pool 或形成单个长事务。
- `REDIS_ADDR` / `REDIS_PASSWORD` / `REDIS_DB`：主机上的 Redis，用于验证码、session cache、API token 限流。
- `REDIS_POOL_SIZE` / `REDIS_MIN_IDLE_CONNS` / `REDIS_*_TIMEOUT`：go-redis 连接池与 dial/read/write/pool wait timeout；默认 `0` 沿用 go-redis 默认值，高 QPS 部署按实例容量和 `/v1` 峰值并发调优。
- `SESSION_SECRET` / `SESSION_AUTH_CACHE_TTL_SECONDS` / `SESSION_LAST_SEEN_UPDATE_INTERVAL_SECONDS`：登录 session hash secret、portal session 用户短缓存 TTL、`sessions.last_seen_at` 写入节流窗口。
- `LOG_LEVEL`：后端日志级别，支持 `trace`、`debug`、`info`、`warn`、`error`、`fatal`；本地排查可用 `LOG_LEVEL=debug make backend-dev`。
- `ENABLE_PPROF` / `ENABLE_METRICS` / `OBSERVABILITY_ADDR`：可选只读诊断端点；默认关闭并绑定 `127.0.0.1:6060`，启用后提供 `/debug/pprof/*`、`/debug/pprof/trace` 与 `/debug/vars`，不要绑定公网地址。
- `MAIL_DRIVER`：邮箱验证码驱动，支持 `smtp`、`postal`、`log`。
- `MAIL_SEND_TIMEOUT_SECONDS`：SMTP/Postal 单次发信超时，默认 10 秒，避免邮件服务卡顿长期阻塞发码请求。
- `SMTP_*`：SMTP 驱动配置；`SMTP_FROM` / `SMTP_FROM_NAME` 也作为 Postal 发件人。
- `POSTAL_API_URL` / `POSTAL_API_KEY`：Postal HTTP API 驱动配置。
- `API_TOKEN_PEPPER`：API token hash pepper，数据库只存 `sha256(token + "." + pepper)`。
- `API_PREAUTH_IP_*` / `API_TOKEN_AUTH_CACHE_TTL_SECONDS` / `API_LAST_USED_UPDATE_INTERVAL_SECONDS` / `MAX_ACTIVE_TOKENS_PER_USER`：`/v1` pre-auth IP 粗限流、token auth 短缓存、`last_used_at` 写入节流与单账号 active token 数量上限。
- `API_REQUEST_LOG_QUEUE_SIZE` / `API_REQUEST_LOG_BATCH_SIZE` / `API_REQUEST_LOG_FLUSH_INTERVAL` / `API_REQUEST_LOG_RETENTION_DAYS`：`/v1` request log 有界队列、批量写入间隔与 raw log 保留天数；dashboard 统计读取聚合表。
- `ENABLE_SYNC_WORKER`：API 进程是否启动后台同步，默认 `false`；生产建议保持关闭并使用独立 worker。
- `SYNC_INTERVAL_MINUTES` / `SYNC_FULL_INTERVAL_HOURS` / `SYNC_RUN_FINISH_TIMEOUT`：仅在 `ENABLE_SYNC_WORKER=true` 时控制 API 进程内 incremental/full 同步周期，以及请求取消后持久化 sync run 结束状态的最大等待时间。

前端复制：

```bash
cp frontend/.env.example frontend/.env
```

## 本地启动

先确保主机上的 PostgreSQL 与 Redis 已启动，并在 `backend/.env` 中配置 `DATABASE_DSN`、`REDIS_ADDR`。如果后端运行在 Docker Compose 容器内，连接主机服务时通常使用 `host.docker.internal`，例如 `REDIS_ADDR=host.docker.internal:6379`，`DATABASE_DSN` 中的主机名同理改为 `host.docker.internal`。

```bash
make dev
```

或分别启动：

```bash
make backend-dev
make frontend-dev
```

需要输出 debug 日志时，对启动该 Go 进程的命令设置 `LOG_LEVEL`；单独运行同步命令时也要单独设置，或写入 `backend/.env`：

```bash
LOG_LEVEL=debug make backend-dev
LOG_LEVEL=debug make sync
```

macOS 的默认临时目录路径可能过长，Nuxt dev 的 Vite Node IPC socket 会因此触发 `connect EINVAL`。`make frontend-dev` 会把 `TMPDIR` 固定到 `/tmp`；若直接进入 `frontend` 运行，请使用 `TMPDIR=/tmp pnpm dev`。

## 性能验证

只读基线流程见 `docs/performance.md`。常用命令：

```bash
make bench
DATABASE_DSN='postgres://...' make perf-explain
SOURCE_DATABASE_DSN='postgres://readonly_user:...' make perf-explain-source
make frontend-analyze
```

`REDIS_BENCH_ADDR=localhost:6379 REDIS_BENCH_DB=15 make bench` 会额外跑真实 Redis 限流 benchmark；未设置专用 Redis DB 时该 benchmark 自动跳过。


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

`SOURCE_DATABASE_DSN` 必须使用主库只读账号，且生产环境默认只放在独立 sync worker 的 env 中。本项目禁止修改主项目数据库。同步任务按 source `patch.id` keyset 分页读取，incremental 查询拆分 `updated` 与 `resource_update_time` 条件，clean DB 写入按批提交；full sync 通过 `sync_run_seen` staging 表标记本次见过的 source patch，再在所有批次成功后统一标记未见条目为 deleted。

## 创建管理员方式

先通过前端注册账号，然后在本项目独立库中执行：

```sql
UPDATE users SET is_admin = true WHERE email = 'admin@example.com';
```

不要复制或复用主站账号权限。

## 邮箱驱动配置

通过 `MAIL_DRIVER` 选择邮箱验证码驱动：`smtp` 使用 `SMTP_HOST`、`SMTP_PORT`、`SMTP_USERNAME`、`SMTP_PASSWORD`、`SMTP_FROM`、`SMTP_FROM_NAME`；`postal` 使用 `POSTAL_API_URL`、`POSTAL_API_KEY`，并复用 `SMTP_FROM`、`SMTP_FROM_NAME` 作为发件人。SMTP 与 Postal 单次发信都受 `MAIL_SEND_TIMEOUT_SECONDS` 限制。`POSTAL_API_URL` 必须使用 `https://`，可填 Postal 根地址或完整 `/api/v1/send/message` 地址。`log` 会将验证码写入结构化日志。开发环境未配置 SMTP 时，后端仍会回退到日志驱动；生产应选择并配置真实邮件驱动。

## API token 申请流程

1. 用户通过邮箱验证码注册或登录。
2. 登录后访问 `/apply` 提交一次账户级申请：负责人、项目地址、预估请求量、使用场景、同意条款。
3. 管理员在 `/admin/applications` 审核账户申请。
4. 申请 `approved` 后，该账户可在 `/dashboard/tokens` 创建 token；active token 数量受 `MAX_ACTIVE_TOKENS_PER_USER` 限制（默认 10），管理员账户默认视为已通过申请。
5. token 明文只返回一次；数据库只保存 hash。
6. 用户可随时删除 token；删除会直接移除数据库中的 token 记录，该 token 不能继续访问 `/v1`。

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

- `deploy/docker-compose.yml` 只编排 backend/frontend；PostgreSQL、Redis 使用主机服务。
- 生产同样使用外部 PostgreSQL、Redis；主库只读 SOURCE DB 账号只配置给独立 sync worker。
- 生产默认将 API HTTP 与 sync worker 分离：API 保持 `ENABLE_SYNC_WORKER=false`，使用 k8s CronJob、systemd timer 或系统 cron 执行 `touchgal-sync --mode=incremental`；k8s CronJob 应设置 `concurrencyPolicy: Forbid`，服务层 PostgreSQL advisory lock 作为跨进程兜底；只有小数据量或本地调试才建议启用 API 进程内 scheduler。
- 推荐 nginx/Ingress 终止 TLS，并将 `SESSION_COOKIE_SECURE=true`。

## 安全注意事项

- 登录 session 只通过 HttpOnly Cookie 保存，前端不写 localStorage。
- API token 明文只显示一次，日志不得记录明文 token。
- 邮箱验证码只存 hash，并有 TTL、冷却和最大尝试次数。
- 管理接口必须 `users.is_admin=true`。
- 公开 API 默认只返回 SFW 条目。
- 响应错误不暴露数据库结构。
- 主机 PostgreSQL/Redis 只监听可信接口，并通过 PostgreSQL 用户权限、Redis 密码或本机防火墙限制访问；容器通过 `host.docker.internal` 连接主机服务时不应暴露无认证 Redis 到公网。
- `/v1` pre-auth IP 限流只信任来自 loopback/private/link-local peer 的 `X-Forwarded-For` / `X-Real-IP`；生产应让后端只暴露给可信反向代理或内网入口。
- `/v1` Redis 限流同时按 token、user、application 独立计数，账号级/应用级上限不会被多 token 放大。
- clean DB 不包含主项目 user/email/password/IP/role/session/token/resource link。
