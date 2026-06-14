# 部署文档

本文面向生产或类生产环境部署。默认拓扑：PostgreSQL 与 Redis 由宿主机或托管服务提供，`deploy/docker-compose.yml` 只启动 API 后端与 Nuxt 前端；同步任务默认独立运行，不嵌入 API 进程。

## 1. 确认部署拓扑

推荐拓扑：

```text
Client
  -> TLS reverse proxy / load balancer
  -> Nuxt frontend
  -> Go API backend
  -> clean PostgreSQL + Redis

TouchGal main PostgreSQL (read-only)
  -> independent sync worker
  -> clean PostgreSQL
```

关键决策：

- API 进程只需要 clean DB 与 Redis。生产默认保持 `ENABLE_SYNC_WORKER=false`。
- 独立 sync worker 才需要 `SOURCE_DATABASE_DSN`。不要把主站只读 DSN 放进不需要同步的 API 进程环境。
- `/v1/ready` 只检查 clean PostgreSQL 与 Redis，用于 API readiness；它不连接 source DB。
- 反向代理可以把同源 `/api/` 转发到后端根路径，也可以让前端直接访问独立 API 域名。选择跨域部署时必须同时配置 CORS、Cookie domain、CSP `connect-src` 与前端 `NUXT_PUBLIC_API_BASE_URL`。

## 2. 准备主机与依赖

主机需要：

- Linux 或其他可长期运行 Docker/systemd/Kubernetes workload 的环境。
- PostgreSQL clean DB。
- Redis。
- Docker Engine 与 Docker Compose，或 Go/Node 构建环境。
- 可选：`goose` CLI，用于显式执行迁移；API 与 sync 二进制启动时也会应用嵌入式迁移。
- 可选：nginx、Caddy、Traefik 或云负载均衡，用于 TLS、HSTS、请求体限制、反向代理与静态缓存。

生产建议：

- 后端与 sync worker 使用非 root 用户运行。
- 容器保持 read-only rootfs、drop capabilities、`no-new-privileges`。
- systemd 使用专用 `touchgal-api` 用户/组，并保留 `deploy/systemd-touchgal-api.service` 中的 sandboxing 设置。
- Kubernetes CronJob 保留 `concurrencyPolicy: Forbid` 与 `securityContext`。

## 3. 创建 clean PostgreSQL 数据库

clean DB 是本项目唯一的公开查询数据源。它不能与 TouchGal 主站 schema 混用。

示例步骤：

1. 为本项目生成独立数据库密码。
2. 以 PostgreSQL 管理员连接数据库服务器。
3. 创建独立登录角色与数据库。

```sql
CREATE ROLE touchgal_api LOGIN PASSWORD 'replace-with-a-generated-password';
CREATE DATABASE touchgal_api OWNER touchgal_api;
```

- `DATABASE_DSN` 指向此 clean DB。
- API 与 sync worker 都写 clean DB，但 sync worker 使用 `SYNC_DB_*` 独立连接池，避免大批量同步占用 API pool。
- 迁移由 `backend/internal/db/migrations` 管理；不要手工修改已应用迁移。
- 数据库备份应覆盖 clean DB、`api_tokens` hash、应用审核状态、session 表、同步记录和 dashboard 聚合表。

## 4. 准备 source PostgreSQL 只读账号

source DB 是 TouchGal 主站数据库。它只允许同步进程读取公开元数据所需表。

要求：

- `SOURCE_DATABASE_DSN` 使用只读账号。
- 禁止授予写权限、`CREATE`、`TEMPORARY`、DDL 权限或 PostgreSQL 17+ table `MAINTAIN` 权限。
- 同步启动前会执行 source 只读校验；校验失败时不得绕过。
- 只在独立 sync worker 的环境文件或 Kubernetes Secret 中保存 `SOURCE_DATABASE_DSN`。

## 5. 准备 Redis

Redis 用于：

- session 短 TTL cache。
- 验证码和 API pre-auth 限流。
- API token 授权缓存撤销版本。
- token、user、application 三个维度的 `/v1` 限流计数。

要求：

- `REDIS_ADDR` 指向独立 Redis 或托管 Redis。
- 生产 Redis 需要认证时配置 `REDIS_PASSWORD`。
- 多实例 API 必须共享同一个 Redis，否则限流和 token 撤销不会跨实例生效。
- 高 QPS 部署优先调整 `REDIS_POOL_SIZE`、`REDIS_MIN_IDLE_CONNS`、`REDIS_*_TIMEOUT`，不要在代码或部署脚本里创建额外 Redis client。

## 6. 生成生产密钥

生产必须使用非默认且至少 32 字节的随机密钥：

```bash
openssl rand -base64 48
```

至少生成两份不同值：

- `SESSION_SECRET`：签名/保护服务端 session。
- `API_TOKEN_PEPPER`：参与 API token hash。

要求：

- `APP_ENV=production` 时默认密钥会导致启动失败。
- `SESSION_COOKIE_SECURE=true` 是生产必需项。
- 不要把 `SESSION_SECRET`、`API_TOKEN_PEPPER`、API token 明文、OTP、DSN 写入日志、issue、PR 或前端代码。

## 7. 准备后端环境文件

从模板开始：

```bash
cp backend/.env.example backend/.env
```

生产 API 进程的 `backend/.env` 至少要改：

- `APP_ENV=production`
- `DATABASE_DSN`
- `REDIS_ADDR`
- `SESSION_SECRET`
- `SESSION_COOKIE_NAME`
- `SESSION_COOKIE_DOMAIN`
- `SESSION_COOKIE_SECURE=true`
- `MAIL_DRIVER` 与对应 SMTP/Postal 配置
- `TURNSTILE_SECRET_KEY`
- `API_TOKEN_PEPPER`
- `PUBLIC_BASE_URL`
- `API_BASE_URL`
- `ENABLE_SYNC_WORKER=false`

独立 sync worker 使用另一份环境文件或 Secret：

- 继承 clean DB、`SYNC_DB_*`、`SOURCE_DB_*`、日志和同步配置。
- 增加 `SOURCE_DATABASE_DSN`。
- 不需要 `SESSION_SECRET`、邮件、Turnstile、Cookie 等仅 API/portal 使用的配置，但保留这些配置也不会改变 sync 行为。

## 8. 准备前端环境文件

从模板开始：

```bash
cp frontend/.env.example frontend/.env
```

常见生产配置：

- 同源反代（浏览器走同源，SSR 可直连后端）：`NUXT_PUBLIC_API_BASE_URL=/api`，`NUXT_API_BASE_URL=http://backend:8080`（Docker Compose）或 `http://127.0.0.1:8080`（同机 systemd）。
- 同源绝对 URL：`NUXT_PUBLIC_API_BASE_URL=https://developer.example.com/api`
- 独立 API origin：`NUXT_PUBLIC_API_BASE_URL=https://api.example.com`
- Turnstile：`NUXT_PUBLIC_TURNSTILE_SITE_KEY=<Cloudflare Turnstile site key>`

`useApi()` 浏览器端使用 `NUXT_PUBLIC_API_BASE_URL`；SSR 阶段优先使用 server-only 的 `NUXT_API_BASE_URL`，并继续转发请求 Cookie。若 `NUXT_PUBLIC_API_BASE_URL` 是 `/api` 这类相对路径，生产必须配置绝对的 `NUXT_API_BASE_URL`，且不能从请求 `Host` 推导 SSR fetch origin；未配置时 SSR 认证会 fail closed，避免把 Cookie 转发到伪造 Host。如果使用 `deploy/nginx.conf` 的 `/api/` location，`proxy_pass http://touchgal_backend/;` 会去掉 `/api/` 前缀，因此前端请求 `/api/auth/me` 会到达后端 `/auth/me`。

## 9. 构建与发布制品

### Docker Compose 路径

Compose 示例构建后端和前端镜像：

```bash
docker compose -f deploy/docker-compose.yml build
```

注意：

- Compose 文件不包含 PostgreSQL 和 Redis；它们需要在宿主机或外部服务中运行。
- 后端容器访问宿主机 PostgreSQL/Redis 时，DSN/Redis 地址使用 `host.docker.internal`。
- Compose 示例为后端和前端设置了非 root 用户、read-only rootfs、tmpfs、drop capabilities、`no-new-privileges`、CPU/memory/pid 限制。

### 二进制/systemd 路径

构建后端二进制：

```bash
cd backend
go build -o /usr/local/bin/touchgal-api ./cmd/api
go build -o /usr/local/bin/touchgal-sync ./cmd/sync
```

部署 systemd 服务时：

- 创建 `touchgal-api` 系统用户与组。
- 将后端工作目录放到 `/opt/touchgal-developer/backend` 或同步修改 service 文件。
- 将环境文件放到 service 的 `EnvironmentFile` 指向路径。
- 保留 `NoNewPrivileges`、`ProtectSystem`、`PrivateTmp`、`CapabilityBoundingSet=` 等 sandboxing 设置。

前端生产构建由 `frontend/Dockerfile` 完成；二进制部署路径可在构建机执行 `pnpm build` 后发布 `.output`，运行入口是 `node .output/server/index.mjs`。

## 10. 执行迁移

推荐在启动新版本前显式迁移：

```bash
DATABASE_DSN='postgres://touchgal_api:password@db-host:5432/touchgal_api?sslmode=require' make migrate-up
```

说明：

- 该命令需要 `goose` CLI。
- API 与 sync 二进制启动时也会调用嵌入式迁移；显式迁移的价值是把 schema 变更放到可控窗口。
- 迁移只作用于 clean DB。

## 11. 初始化 clean DB 数据

首次上线或 clean DB 重建后执行 full sync：

```bash
cd backend
DATABASE_DSN='postgres://touchgal_api:password@db-host:5432/touchgal_api?sslmode=require' \
SOURCE_DATABASE_DSN='postgres://readonly_user:password@main-db-host:5432/touchgal?sslmode=require' \
go run ./cmd/sync --mode=full
```

使用 Compose 镜像时可运行：

```bash
docker compose -f deploy/docker-compose.yml run --rm \
  -e SOURCE_DATABASE_DSN='postgres://readonly_user:password@main-db-host:5432/touchgal?sslmode=require' \
  backend touchgal-sync --mode=full
```

要求：

- full sync 会记录本次看见的 source patch，并且只有所有批次成功后才把未见条目标记 deleted。
- full sync 应放在低峰窗口，source 账号必须只读。
- 如果 full sync 失败，先修复失败原因再重跑；不要手工标记 deleted。

## 12. 启动 API 与前端

Compose 启动：

```bash
docker compose -f deploy/docker-compose.yml up --build -d
```

API 启动后会：

1. 加载 `backend/.env`。
2. 校验生产密钥、Cookie secure、Turnstile、邮件、timeout、Redis、观测地址等配置。
3. 连接 clean PostgreSQL。
4. 应用嵌入式迁移。
5. 连接 Redis。
6. 组装 repository、service、router 和 request log writer。
7. 监听 `HTTP_ADDR`。

前端启动后会读取 Nuxt public runtime config，页面请求统一通过 `useApi()` 发送，且保留 `credentials: 'include'`。

## 13. 配置反向代理与 TLS

`deploy/nginx.conf` 是完整示例，包含：

- `/api/` 转发到后端根路径。
- `client_max_body_size 128k` 与请求体/代理超时。
- 按 IP 限制请求速率与并发连接数。
- HSTS、CSP、`frame-ancestors`、`X-Frame-Options`、`Referrer-Policy`、`Permissions-Policy`、`X-Content-Type-Options`。
- `/_nuxt/` immutable 长缓存。
- `/logo.webp` 短 TTL。
- `/api/` 返回 `Cache-Control: no-store`。

配置要求：

- TLS 证书在反向代理或负载均衡层终止。
- 如果 `NUXT_PUBLIC_API_BASE_URL` 指向独立 API origin，CSP 的 `connect-src` 必须加入该 origin。
- 如果 API 与前端跨子域，`SESSION_COOKIE_DOMAIN` 应覆盖 portal 与 API 域名，并确认浏览器 SameSite 与 secure 策略符合部署方式。

## 14. 安排增量同步

推荐使用独立 worker 周期运行 incremental sync。

### Kubernetes CronJob

使用 `deploy/k8s-cronjob-sync.yaml`：

- `command: ["touchgal-sync", "--mode=incremental"]`
- `concurrencyPolicy: Forbid`
- `activeDeadlineSeconds: 3600`
- `restartPolicy: OnFailure`
- `readOnlyRootFilesystem: true`
- `capabilities.drop: ["ALL"]`
- `envFrom.secretRef` 指向包含 `DATABASE_DSN`、`SOURCE_DATABASE_DSN`、`SYNC_DB_*`、`SOURCE_DB_*` 的 Secret。

### systemd timer

使用独立 oneshot service 运行 `touchgal-sync --mode=incremental`，并用 timer 定期触发。API service 继续保持 `ENABLE_SYNC_WORKER=false`。

要求：

- 数据库层仍有 PostgreSQL advisory lock 防止跨入口并发同步。
- CronJob/timer 层也要避免重叠运行；不要依赖数据库锁作为常态调度器。
- full sync 保持人工触发或低峰维护任务。

## 15. 验证上线结果

后端健康检查：

```bash
curl -fsS http://127.0.0.1:8080/v1/health
curl -fsS http://127.0.0.1:8080/v1/ready
```

反向代理检查：

```bash
curl -fsSI https://developer.example.com/
curl -fsS https://developer.example.com/api/v1/ready
```

关键检查项：

- `/v1/ready` 返回 `{ "success": true, "data": { "status": "ready", "version": "v1" } }`。
- 前端页面能加载 `/_nuxt/` 资源。
- 登录验证码能通过真实 SMTP 或 Postal 发出。
- Turnstile 站点 key 与 secret key 匹配。
- `/auth/*`、已登录 portal API 与 `/admin/*` 响应带 `Cache-Control: no-store`。
- `/v1` 未带 token 时被拒绝，带有效 token 时受 Redis 限流保护。
- sync worker 日志显示 incremental 或 full sync 完成。
- 日志不包含 API token 明文、OTP、session token、DSN、pepper。

## 16. 回滚与维护

- 应用版本回滚：回滚镜像或二进制，再重启 API/前端；不要回滚 clean DB 数据文件。
- 迁移回滚：只在确认 Down migration 安全且没有新代码依赖新 schema 时执行；生产优先前滚修复。
- token 泄露：删除对应 `api_tokens` 记录；服务会 bump Redis 共享撤销版本。
- source DB 权限异常：停止 sync worker，修复只读账号权限后再恢复。
- Redis 故障：API token auth、rate limit 与 session cache 都受影响；恢复同一个 Redis 数据面，避免多实例状态分裂。
- raw request log 保留由 `API_REQUEST_LOG_RETENTION_DAYS` 控制；dashboard 聚合表按查询窗口保留。

## 17. 环境变量列表

### 后端基础变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `APP_ENV` | `development` | 生产设为 `production` | 控制生产校验、日志输出形态与 `MAIL_DRIVER=log` 是否允许。 |
| `LOG_LEVEL` | `info` | 建议 `info` | 支持 `trace`、`debug`、`info`、`warn`、`error`、`fatal`。排障可临时设为 `debug`，不要记录 secret。 |
| `HTTP_ADDR` | `:8080` | 按监听方式设置 | API HTTP server 监听地址。容器内通常保持 `:8080`。 |
| `HTTP_READ_HEADER_TIMEOUT` | `10s` | 必须为正 | 读取请求头超时，抵御慢客户端。 |
| `HTTP_READ_TIMEOUT` | `15s` | 必须为正 | 读取完整请求超时。 |
| `HTTP_WRITE_TIMEOUT` | `60s` | 必须为正且大于 `HTTP_READ_TIMEOUT + DB_QUERY_TIMEOUT` | 写响应超时。需给数据库超时错误留出返回窗口。 |
| `HTTP_IDLE_TIMEOUT` | `120s` | 必须为正 | keep-alive 空闲连接超时。 |
| `HTTP_MAX_HEADER_BYTES` | `1048576` | 必须为正 | HTTP 请求头最大字节数。 |
| `PUBLIC_BASE_URL` | `http://localhost:3000` | 设置为门户公网 URL | 后端生成面向用户的链接时使用。 |
| `API_BASE_URL` | `http://localhost:8080` | 设置为 API 公网 URL 或同源 API base | 后端公开自身 API 地址时使用。 |

### 后端观测变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `ENABLE_PPROF` | `false` | 默认关闭 | 开启 `/debug/pprof/*` 和 trace。为 `true` 时 `OBSERVABILITY_ADDR` 只能是 localhost 或 loopback。 |
| `ENABLE_METRICS` | `false` | 按需开启 | 开启 `/debug/vars` expvar 指标。仅 metrics 时可绑定 private 管理地址。 |
| `OBSERVABILITY_ADDR` | `127.0.0.1:6060` | 不得绑定公网或 wildcard | 诊断 HTTP server 监听地址。通过 SSH tunnel、kubectl port-forward 或内网管理面访问。 |

### clean DB 变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `DATABASE_DSN` | `postgres://touchgal_api:touchgal_api@localhost:5432/touchgal_api?sslmode=disable` | 必填，使用生产 clean DB 与安全 TLS 策略 | API 与 sync worker 写入/读取本项目 clean DB。 |
| `DB_POOL_MAX_CONNS` | `16` | 必须大于 0 | API clean DB pool 最大连接数。 |
| `DB_POOL_MIN_CONNS` | `1` | 不能为负，不能大于 max | API clean DB pool 最小连接数。 |
| `DB_POOL_MIN_IDLE_CONNS` | `0` | 不能为负，不能大于 max | API clean DB pool 最小空闲连接数。 |
| `DB_POOL_MAX_CONN_LIFETIME` | `1h` | 按数据库/代理策略调整 | API pool 单连接最大生命周期。 |
| `DB_POOL_MAX_CONN_IDLE_TIME` | `15m` | 按数据库/代理策略调整 | API pool 空闲连接最大保留时间。 |
| `DB_POOL_HEALTH_CHECK_PERIOD` | `1m` | 必须为正 | API pool 健康检查周期。 |
| `DB_STATEMENT_TIMEOUT` | `30s` | 可为 `0` 禁用，不建议生产禁用 | API 连接设置 PostgreSQL `statement_timeout`。 |
| `DB_IDLE_IN_TRANSACTION_SESSION_TIMEOUT` | `1m` | 可为 `0` 禁用，不建议生产禁用 | API 连接设置 PostgreSQL idle transaction 超时。 |
| `DB_QUERY_TIMEOUT` | `35s` | 可为 `0` 禁用；若为正需小于 `HTTP_WRITE_TIMEOUT - HTTP_READ_TIMEOUT` | repository query context timeout。 |

### sync 写 clean DB 变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `SYNC_DB_POOL_MAX_CONNS` | `4` | 必须大于 0 | sync worker 写 clean DB 的独立 pool 最大连接数。 |
| `SYNC_DB_POOL_MIN_CONNS` | `0` | 不能为负，不能大于 max | sync target pool 最小连接数。 |
| `SYNC_DB_POOL_MIN_IDLE_CONNS` | `0` | 不能为负，不能大于 max | sync target pool 最小空闲连接数。 |
| `SYNC_DB_POOL_MAX_CONN_LIFETIME` | `1h` | 按数据库/代理策略调整 | sync target pool 单连接最大生命周期。 |
| `SYNC_DB_POOL_MAX_CONN_IDLE_TIME` | `15m` | 按数据库/代理策略调整 | sync target pool 空闲连接最大保留时间。 |
| `SYNC_DB_POOL_HEALTH_CHECK_PERIOD` | `1m` | 必须为正 | sync target pool 健康检查周期。 |
| `SYNC_DB_STATEMENT_TIMEOUT` | `15m` | 可为 `0` 禁用 | sync 写 clean DB 的 PostgreSQL statement timeout。 |
| `SYNC_DB_IDLE_IN_TRANSACTION_SESSION_TIMEOUT` | `5m` | 可为 `0` 禁用 | sync 写 clean DB 的 idle transaction 超时。 |
| `SYNC_DB_QUERY_TIMEOUT` | `16m` | 可为 `0` 禁用 | sync repository query context timeout。 |

### source DB 变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `SOURCE_DATABASE_DSN` | 空或 `postgres://readonly_user:password@main-db-host:5432/touchgal?sslmode=require` | 独立 sync worker 必填；API 进程默认不配置 | TouchGal 主站只读数据库 DSN。必须只读。 |
| `SOURCE_DB_POOL_MAX_CONNS` | `4` | 必须大于 0 | source DB pool 最大连接数。 |
| `SOURCE_DB_POOL_MIN_CONNS` | `0` | 不能为负，不能大于 max | source DB pool 最小连接数。 |
| `SOURCE_DB_POOL_MIN_IDLE_CONNS` | `0` | 不能为负，不能大于 max | source DB pool 最小空闲连接数。 |
| `SOURCE_DB_POOL_MAX_CONN_LIFETIME` | `1h` | 按 source DB 策略调整 | source DB pool 单连接最大生命周期。 |
| `SOURCE_DB_POOL_MAX_CONN_IDLE_TIME` | `15m` | 按 source DB 策略调整 | source DB pool 空闲连接最大保留时间。 |
| `SOURCE_DB_POOL_HEALTH_CHECK_PERIOD` | `1m` | 必须为正 | source DB pool 健康检查周期。 |
| `SOURCE_DB_STATEMENT_TIMEOUT` | `15m` | 可为 `0` 禁用 | source DB PostgreSQL statement timeout。 |
| `SOURCE_DB_IDLE_IN_TRANSACTION_SESSION_TIMEOUT` | `1m` | 可为 `0` 禁用 | source DB idle transaction 超时。 |
| `SOURCE_DB_QUERY_TIMEOUT` | `16m` | 可为 `0` 禁用 | source DB query context timeout。 |

### Redis 变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `REDIS_ADDR` | `localhost:6379` | 必填 | Redis 地址。Compose 容器访问宿主机可用 `host.docker.internal:6379`。 |
| `REDIS_PASSWORD` | 空 | 按 Redis 认证配置 | Redis 密码。 |
| `REDIS_DB` | `0` | 多环境隔离时设置独立 DB | Redis logical DB。生产托管 Redis 可能只允许 `0`。 |
| `REDIS_POOL_SIZE` | `0` | 不能为负 | `0` 使用 go-redis 默认 pool size；高 QPS 可显式设置。 |
| `REDIS_MIN_IDLE_CONNS` | `0` | 不能为负；pool size 为正时不能大于 pool size | Redis 最小空闲连接数。 |
| `REDIS_DIAL_TIMEOUT` | `0` | 不能为负 | `0` 时项目使用固定安全默认值 `5s`。 |
| `REDIS_READ_TIMEOUT` | `0` | 不能为负 | `0` 时项目使用固定安全默认值 `3s`。 |
| `REDIS_WRITE_TIMEOUT` | `0` | 不能为负 | `0` 时项目使用固定安全默认值 `3s`。 |
| `REDIS_POOL_TIMEOUT` | `0` | 不能为负 | `0` 时项目使用固定安全默认值 `4s`。 |

### session 与 Cookie 变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `SESSION_SECRET` | `please-change-this-64-byte-secret` | 必须替换为至少 32 字节随机值 | session secret。默认值在 `APP_ENV=production` 下拒绝启动。 |
| `SESSION_COOKIE_NAME` | `tgal_dev_session` | 建议改成生产专用名称 | 登录 Cookie 名称。 |
| `SESSION_COOKIE_DOMAIN` | 空 | 跨子域部署时设置父域 | Cookie domain。空值表示当前 host。 |
| `SESSION_COOKIE_SECURE` | `false` | 生产必须为 `true` | 只允许 HTTPS 发送 Cookie。 |
| `SESSION_TTL_HOURS` | `720` | 必须为正 | session 有效小时数。 |
| `SESSION_AUTH_CACHE_TTL_SECONDS` | `60` | 必须为正 | Redis session auth cache TTL。 |
| `SESSION_LAST_SEEN_UPDATE_INTERVAL_SECONDS` | `300` | 必须为正 | `sessions.last_seen_at` 最小写入间隔，避免每次请求写 DB。 |

### 邮件变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `MAIL_DRIVER` | `smtp` | 生产使用 `smtp` 或 `postal` | 支持 `smtp`、`postal`、`log`。`log` 仅允许 `APP_ENV=development`。 |
| `SMTP_HOST` | `smtp.example.com` | `MAIL_DRIVER=smtp` 且生产时必填 | SMTP server host。 |
| `SMTP_PORT` | `587` | 按 SMTP 服务设置 | SMTP server port。 |
| `SMTP_USERNAME` | 空 | 按 SMTP 服务设置 | SMTP 用户名。 |
| `SMTP_PASSWORD` | 空 | 按 SMTP 服务设置 | SMTP 密码。 |
| `SMTP_FROM` | `no-reply@example.com` | `MAIL_DRIVER=smtp` 且生产时必填 | 发件地址。 |
| `SMTP_FROM_NAME` | `TouchGal API` | 按品牌设置 | 发件显示名。 |
| `MAIL_SEND_TIMEOUT_SECONDS` | `10` | 必须为正 | 发送邮件超时。 |
| `POSTAL_API_URL` | `https://postal.example.com` | `MAIL_DRIVER=postal` 时必填且必须为 HTTPS 绝对 URL | Postal API URL。 |
| `POSTAL_API_KEY` | 空 | `MAIL_DRIVER=postal` 时必填 | Postal API key。 |

### 验证码与 Turnstile 变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `AUTH_CODE_IP_MINUTE_LIMIT` | `20` | 必须为正 | 单 IP 每分钟发码限制。 |
| `AUTH_CODE_IP_DAILY_LIMIT` | `200` | 必须为正 | 单 IP 每日发码限制。 |
| `AUTH_CODE_EMAIL_MINUTE_LIMIT` | `3` | 必须为正 | 单邮箱每分钟发码限制。 |
| `AUTH_CODE_EMAIL_DAILY_LIMIT` | `20` | 必须为正 | 单邮箱每日发码限制。 |
| `AUTH_CODE_IP_EMAIL_MINUTE_LIMIT` | `3` | 必须为正 | IP+email 组合每分钟发码限制。 |
| `AUTH_CODE_IP_EMAIL_DAILY_LIMIT` | `10` | 必须为正 | IP+email 组合每日发码限制。 |
| `EMAIL_CODE_TTL_MINUTES` | `10` | 必须为正 | 邮箱验证码有效分钟数。 |
| `EMAIL_CODE_RESEND_COOLDOWN_SECONDS` | `60` | 必须为正 | 同一邮箱重发冷却。 |
| `EMAIL_CODE_MAX_ATTEMPTS` | `5` | 必须为正 | 验证码最大尝试次数。 |
| `TURNSTILE_SECRET_KEY` | 空 | `APP_ENV=production` 时必填 | Cloudflare Turnstile secret key。开发留空会跳过人机验证。 |

### API token 与 `/v1` 防护变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `API_TOKEN_PEPPER` | `please-change-this-long-random-secret` | 必须替换为至少 32 字节随机值 | API token hash pepper。默认值在 `APP_ENV=production` 下拒绝启动。 |
| `API_TOKEN_PREFIX` | `tgal_live` | 上线后谨慎修改 | 新建 token 明文前缀。修改后只影响新 token。 |
| `DEFAULT_TOKEN_MINUTE_LIMIT` | `60` | 必须为正 | token 默认每分钟限制。 |
| `DEFAULT_TOKEN_DAILY_LIMIT` | `5000` | 必须为正 | token 默认每日限制。 |
| `API_PREAUTH_IP_MINUTE_LIMIT` | `600` | 必须为正 | `/v1` token 解析前的 IP 每分钟限制。 |
| `API_PREAUTH_IP_DAILY_LIMIT` | `20000` | 必须为正 | `/v1` token 解析前的 IP 每日限制。 |
| `API_TOKEN_AUTH_CACHE_TTL_SECONDS` | `60` | 必须为正 | API token 认证短 TTL cache。每次 cache hit 仍校验 Redis 撤销版本。 |
| `API_TOKEN_AUTH_CACHE_MAX_ENTRIES` | `4096` | 必须为正 | 单进程 token auth cache 最大条目数。 |
| `MAX_ACTIVE_TOKENS_PER_USER` | `10` | 必须为正 | 单用户活跃 token 上限。 |
| `API_LAST_USED_UPDATE_INTERVAL_SECONDS` | `300` | 必须为正 | token last-used 最小写入间隔。 |

### `/v1` request log 变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `API_REQUEST_LOG_QUEUE_SIZE` | `16384` | 必须为正 | API request log writer 有界队列大小。满队列会影响记录完整性，应结合 QPS 调整。 |
| `API_REQUEST_LOG_BATCH_SIZE` | `500` | `1` 到 `5000` | 批量写入 raw log 和聚合表的最大条数。 |
| `API_REQUEST_LOG_FLUSH_INTERVAL` | `1s` | 必须为正 | 批量写入 flush 间隔。 |
| `API_REQUEST_LOG_RETENTION_DAYS` | `14` | 必须为正 | raw `api_request_logs` 保留天数。 |

### 同步变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `ENABLE_SYNC_WORKER` | `false` | API 进程保持 `false` | 是否在 API 进程内启动 sync scheduler。仅本地或小部署可启用。 |
| `SYNC_INTERVAL_MINUTES` | `30` | 必须为正 | API 内置 scheduler 的 incremental 间隔；独立 CronJob 不依赖它。 |
| `SYNC_FULL_INTERVAL_HOURS` | `24` | 必须为正 | API 内置 scheduler 的 full sync 间隔；生产建议 full sync 人工控制。 |
| `SYNC_INCREMENTAL_SAFETY_MINUTES` | `10` | 必须为非负业务安全窗口 | incremental sync 查询回看窗口，减少边界更新时间遗漏。 |
| `SYNC_DEFAULT_CONTENT_POLICY` | `all` | 按公开策略设置 | source 内容限制为空时的默认策略；`all` 表示不额外覆盖。 |
| `SYNC_RUN_FINISH_TIMEOUT` | `15s` | 必须为正 | sync run 完成状态写入的超时。 |

### 站点链接变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `TOUCHGAL_SITE_URL` | `https://www.touchgal.ink` | 按站点设置 | 后端返回或展示 TouchGal 主站链接时使用。 |
| `TOUCHGAL_TECH_DOCS_URL` | `https://github.com/KUN1007/kun-touchgal-next` | 按文档入口设置 | 后端/门户展示技术文档链接时使用。 |
| `API_DOCS_URL` | `/docs/api` | 按门户路由设置 | 后端/门户展示 API 文档入口时使用。 |

### 前端 server runtime 变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `NUXT_API_BASE_URL` | 空 | 生产建议设置为 Nuxt SSR 进程可访问的后端地址，例如 `http://backend:8080` 或 `http://127.0.0.1:8080` | 仅服务端使用。设置后 SSR 请求不依赖 public origin；未设置时回退到 `NUXT_PUBLIC_API_BASE_URL`。 |

### 前端 public runtime 变量

| 变量 | 默认/示例 | 生产要求 | 说明 |
| --- | --- | --- | --- |
| `NUXT_PUBLIC_API_BASE_URL` | `http://localhost:8080` | 浏览器可访问的 API base URL；同源反代可填 `/api`，独立域名可填 `https://api.example.com` | Nuxt public API base URL。`useApi()` 会拼接 endpoint path。 |
| `NUXT_PUBLIC_TOUCHGAL_TECH_DOCS_URL` | `https://github.com/KUN1007/kun-touchgal-next` | 按文档入口设置 | 门户展示技术文档链接。 |
| `NUXT_PUBLIC_API_DOCS_URL` | `/docs/api` | 按门户路由设置 | 门户 API 文档入口。 |
| `NUXT_PUBLIC_TURNSTILE_SITE_KEY` | 空 | 生产配置 Cloudflare site key | 前端 Turnstile site key。必须与后端 secret key 同一站点配置。 |
| `NODE_ENV` | Docker runtime 设置为 `production` | 生产应为 `production` | Nuxt/Node 运行模式；前端 security middleware 在非 production 时允许 dev websocket connect-src。 |

### 性能验证专用变量

这些变量不属于生产服务环境文件，只用于 benchmark 或 EXPLAIN 命令。

| 变量 | 默认/示例 | 说明 |
| --- | --- | --- |
| `REDIS_BENCH_ADDR` | 空 | 设置后才运行真实 Redis rate-limit benchmark。 |
| `REDIS_BENCH_PASSWORD` | 空 | Redis benchmark 使用的密码。 |
| `REDIS_BENCH_DB` | 空 | Redis benchmark 使用的隔离 logical DB；未设置时跳过真实 Redis benchmark。 |
| `keyword` | `summer` | `make perf-explain` 的搜索关键词。 |
| `page` | `1` | `make perf-explain` 的分页页码。 |
| `limit` | `20` | `make perf-explain` 的分页大小。 |
| `days` | `30` | `make perf-explain` 的 dashboard 查询窗口。 |
| `unique_id` | 空 | `make perf-explain` 的游戏详情目标 unique id。 |
| `user_id` | 空 | `make perf-explain` 的 dashboard 用户 id。 |
| `source_last_id` | `0` | `make perf-explain-source` 的 source keyset 起点。 |
| `source_limit` | `1000` | `make perf-explain-source` 的 source page size。 |
| `source_window` | `1 day` | `make perf-explain-source` 的 incremental 回看窗口。 |
