# 本地开发启动文档

本文用于在本机启动 TouchGal API 后端、sync worker 与 Nuxt 开发者门户。默认假设 PostgreSQL 与 Redis 运行在宿主机，后端和前端直接以开发命令启动。

## 1. 准备本地工具

需要安装：

- Go `1.26.4`，与 `backend/go.mod` 一致。
- Node.js `22`。
- Corepack 与 `pnpm@11.5.0`，与 `frontend/package.json` 一致。
- PostgreSQL。
- Redis。
- 可选：`goose` CLI，用于手动执行 `make migrate-up`。
- 可选：Docker Compose，用于容器化启动 backend/frontend。

检查命令：

```bash
go version
node --version
corepack --version
pnpm --version
psql --version
redis-cli --version
```

启用 pnpm：

```bash
corepack enable
```

## 2. 创建本地 clean DB

clean DB 是本项目后端读写的数据库。本地可以使用独立库名与独立用户。

```sql
CREATE ROLE touchgal_api LOGIN PASSWORD 'touchgal_api';
CREATE DATABASE touchgal_api OWNER touchgal_api;
```

本地 DSN：

```text
postgres://touchgal_api:touchgal_api@localhost:5432/touchgal_api?sslmode=disable
```

说明：

- 该库只用于 TouchGal API clean schema。
- 不要把 TouchGal 主站库当成 `DATABASE_DSN`。
- API 和 sync 启动时会自动应用嵌入迁移；也可以用 `make migrate-up` 手动迁移。

## 3. 启动本地 Redis

使用系统服务、Docker 或包管理器启动 Redis。默认后端配置读取：

```env
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

开发环境可以使用 Redis logical DB `0`；如果同机有其他项目，改成独立 DB 编号。

## 4. 准备后端环境文件

复制模板：

```bash
cp backend/.env.example backend/.env
```

本地开发建议配置：

```env
APP_ENV=development
LOG_LEVEL=debug
HTTP_ADDR=:8080
PUBLIC_BASE_URL=http://localhost:3000
API_BASE_URL=http://localhost:8080
DATABASE_DSN=postgres://touchgal_api:touchgal_api@localhost:5432/touchgal_api?sslmode=disable
REDIS_ADDR=localhost:6379
SESSION_SECRET=please-change-this-64-byte-secret
SESSION_COOKIE_NAME=tgal_dev_session
SESSION_COOKIE_SECURE=false
MAIL_DRIVER=log
SMTP_HOST=
TURNSTILE_SECRET_KEY=
API_TOKEN_PEPPER=please-change-this-long-random-secret
ENABLE_SYNC_WORKER=false
```

说明：

- `MAIL_DRIVER=log` 只允许 `APP_ENV=development`，验证码会写日志，不会真正发送邮件。
- `TURNSTILE_SECRET_KEY` 留空时开发环境跳过人机验证。
- 默认 `SESSION_SECRET` 与 `API_TOKEN_PEPPER` 只适合本地开发；生产会拒绝默认值。
- 默认不启用 API 进程内 sync worker。需要同步数据时单独运行 `make sync` 或 `make sync-full`。

## 5. 准备前端环境文件

复制模板：

```bash
cp frontend/.env.example frontend/.env
```

本地开发建议配置：

```env
NUXT_PUBLIC_API_BASE_URL=http://localhost:8080
NUXT_API_BASE_URL=http://localhost:8080
NUXT_PUBLIC_TOUCHGAL_TECH_DOCS_URL=https://github.com/KUN1007/kun-touchgal-next
NUXT_PUBLIC_API_DOCS_URL=/docs/api
NUXT_PUBLIC_TURNSTILE_SITE_KEY=
```

说明：

- Nuxt dev server 默认在 `http://localhost:3000`。
- 后端默认在 `http://localhost:8080`。
- 前端 API 调用统一走 `useApi()`，会带 `credentials: 'include'`。

## 6. 安装前端依赖

```bash
cd frontend
pnpm install --frozen-lockfile
```

后端 Go module 会在 `go run` 或 `go test` 时按 `go.mod` 下载。

## 7. 执行迁移

可选手动迁移：

```bash
DATABASE_DSN='postgres://touchgal_api:touchgal_api@localhost:5432/touchgal_api?sslmode=disable' make migrate-up
```

如果没有安装 `goose`，直接启动 API 也会应用嵌入迁移：

```bash
make backend-dev
```

迁移成功后 clean DB 中会有应用、token、sync、games、tags、companies、request log 与 dashboard 聚合相关表。

## 8. 导入本地数据

没有 source DB 时可以跳过此步骤。API 与门户仍可启动，但公开游戏搜索和详情会返回空结果。

有主站只读 source DB 时，在 `backend/.env` 中配置：

```env
SOURCE_DATABASE_DSN=postgres://readonly_user:password@main-db-host:5432/touchgal?sslmode=require
```

首次导入：

```bash
make sync-full
```

日常增量同步：

```bash
make sync
```

注意：

- source DSN 必须是只读账号。
- sync 会先执行只读权限校验。
- full sync 会在全部批次成功后处理未见条目的 deleted 标记。

## 9. 启动后端 API

```bash
make backend-dev
```

等价于：

```bash
cd backend && go run ./cmd/api
```

启动成功后检查：

```bash
curl -fsS http://localhost:8080/v1/health
curl -fsS http://localhost:8080/v1/ready
```

开发排障：

- 需要更详细日志时设置 `LOG_LEVEL=debug` 或 `LOG_LEVEL=trace`。
- `/v1/health` 不检查依赖，只表示进程可响应。
- `/v1/ready` 检查 clean DB 与 Redis；Redis 或 DB 未启动会返回 not ready。

## 10. 启动 Nuxt 前端

另开终端：

```bash
make frontend-dev
```

等价于：

```bash
cd frontend && TMPDIR="${TMPDIR:-/tmp}" pnpm dev
```

访问：

```text
http://localhost:3000
```

本地登录流程：

1. 打开门户。
2. 输入邮箱请求验证码。
3. 查看后端日志中的验证码。
4. 输入验证码完成登录或注册。
5. 提交 application。
6. 使用管理员门户审核 application 后创建 API token。

## 11. Docker Compose 本地启动

Compose 示例只启动 backend/frontend，PostgreSQL 与 Redis 仍在宿主机。

如果后端运行在容器中，修改 `backend/.env`：

```env
DATABASE_DSN=postgres://touchgal_api:touchgal_api@host.docker.internal:5432/touchgal_api?sslmode=disable
REDIS_ADDR=host.docker.internal:6379
```

启动：

```bash
make dev
```

等价于：

```bash
docker compose -f deploy/docker-compose.yml up --build
```

说明：
- `NUXT_PUBLIC_API_BASE_URL` 是浏览器使用的 public base URL；`NUXT_API_BASE_URL` 是 Nuxt SSR 进程使用的 server-only base URL。Compose 会默认给 frontend 容器注入 `NUXT_API_BASE_URL=http://backend:8080`；本地浏览器直连仍可在 `frontend/.env` 使用 `NUXT_PUBLIC_API_BASE_URL=http://localhost:8080`。
- Compose backend healthcheck 使用 `http://127.0.0.1:8080/v1/ready`。
- frontend 会等待 backend service healthy。
- Compose 默认设置 read-only rootfs 和 tmpfs；本地调试需要写临时文件时写入 `/tmp`。

## 12. 常用开发命令

| 命令 | 作用 |
| --- | --- |
| `make backend-dev` | 启动 Go API。 |
| `make frontend-dev` | 启动 Nuxt dev server。 |
| `make sync` | 运行一次 incremental sync。 |
| `make sync-full` | 运行一次 full sync。 |
| `make migrate-up` | 使用 goose CLI 对 clean DB 执行迁移。 |
| `make test` | 运行 `cd backend && go test ./...` 与 `cd frontend && pnpm typecheck`。 |
| `make bench` | 运行 Go benchmark。 |
| `make perf-explain` | 对 clean DB 公开查询执行 EXPLAIN 基线。 |
| `make perf-explain-source` | 对 source DB 同步读取执行 EXPLAIN 基线。 |
| `make frontend-analyze` | 运行 Nuxt bundle analysis。 |

## 13. 本地验证清单

后端改动后：

```bash
cd backend && go test ./...
```

前端改动后：

```bash
cd frontend && pnpm typecheck
```

Nuxt config、route 或 SSR-sensitive 改动后：

```bash
cd frontend && pnpm build
```

全量检查：

```bash
make test
```

手动 smoke test：

```bash
curl -fsS http://localhost:8080/v1/health
curl -fsS http://localhost:8080/v1/ready
```

## 14. 常见问题

### `/v1/ready` 返回 not ready

检查：

- `DATABASE_DSN` 是否指向 clean DB。
- PostgreSQL 是否允许当前用户连接。
- Redis 是否启动，`REDIS_ADDR` 是否正确。
- API 启动日志中是否有迁移或连接错误。

### 邮箱验证码没有发送

开发环境推荐 `MAIL_DRIVER=log`，验证码会出现在后端日志。生产或类生产环境使用 `smtp` 或 `postal`，并配置对应 host、from 和认证凭据。

### 前端请求没有带 Cookie

检查：

- 前端是否通过 `useApi()` 调用后端。
- `NUXT_PUBLIC_API_BASE_URL` 是否与后端实际地址一致。
- 跨域部署时 Cookie domain、secure、SameSite 与反向代理 HTTPS 是否匹配。

### sync 失败

检查：

- `SOURCE_DATABASE_DSN` 是否配置在运行 sync 的进程环境中。
- source 账号是否只读且具备所需 SELECT 权限。
- `SYNC_DB_*` 和 `SOURCE_DB_*` timeout 是否足够覆盖当前批量数据。
- 是否已有另一个 sync run 持有 advisory lock。
