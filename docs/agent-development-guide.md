# TouchGal API Agent Development Guide

面向本仓库维护者和 AI agent 的项目专用开发指南。它补充 `AGENTS.md`、`README.md`、OpenAPI 与源码约定，重点记录改动时不能破坏的边界和推荐流程。

## 不可变项目边界

- 本项目是独立的开发者 API 与门户，不直接对外读取 TouchGal 主站 schema。
- 同步层只从主库只读账号读取公开元数据，写入 clean DB 后再由 `/v1` 对外服务。
- 禁止同步或返回主站用户、会话、评论、私信、举报、收藏、文件夹、资源下载链接、上传者 `user_id`、主库内部自增 id。
- clean DB 可以保存 `source_patch_id` 供同步使用，但公开 API 永不返回它。
- API token 明文只在创建响应中返回一次；数据库只保存 `sha256(token + "." + API_TOKEN_PEPPER)`。
- API token 删除/失效必须直接删除 `api_tokens` 记录；不要用 `revoked`/`disabled` 状态保留已失效 token。
- 登录态只走 HttpOnly Cookie；前端不得把 session、API token 或验证码写入 `localStorage`。

## 系统地图

```text
TouchGal main PostgreSQL (read-only)
  -> backend/cmd/sync full|incremental
  -> TouchGal API clean PostgreSQL
  -> backend/cmd/api chi router + services
  -> /v1 API + Nuxt Developer Portal
```

关键入口：

- `backend/cmd/api/main.go`：加载配置、连接 PostgreSQL/Redis、应用嵌入迁移、组装仓储/服务、启动 HTTP server。
- `backend/cmd/sync/main.go`：以 `SOURCE_DATABASE_DSN` 跑 `full` 或 `incremental` 同步。
- `backend/internal/httpserver/server.go`：`chi` route topology 与 middleware 顺序。
- `frontend/nuxt.config.ts`：Nuxt 模块、严格 TypeScript、`runtimeConfig.public`。
- `frontend/composables/useApi.ts`：前端 API 统一入口，负责 `credentials: 'include'` 与 SSR Cookie 转发。

本地调试日志：后端入口读取 `LOG_LEVEL`，支持 `trace`、`debug`、`info`、`warn`、`error`、`fatal`；需要查看 debug 事件时，对启动该 Go 进程的命令设置 `LOG_LEVEL`，例如 `LOG_LEVEL=debug make backend-dev` 或 `LOG_LEVEL=debug make sync`。

## 后端改动流程

### 分层职责

- `handlers`：HTTP 解析、`DecodeJSON`、URL/query 参数、响应映射。
- `services`：输入规范化、业务规则、权限/状态机、组合多个 store。
- `repository`：SQL、pgx row/transaction handling、clean DB 表访问。
- `model`：领域类型、状态常量、sentinel errors。
- `middleware`：request id、recover、CORS、session auth、admin checks、API token auth、rate limiting、request logging。

新增或修改 API 时通常按这个顺序落地：

1. 在 `model` 补充领域类型/status/error。
2. 若需要 schema 变化，新建 `backend/internal/db/migrations/NNNNNN_*.sql`，包含 goose `Up`/`Down`。
3. 在 `repository` 写 SQL；通过 `repository.Queryer` 支持 pool/tx/test fake 注入。
4. 在 `services` 写规范化和业务规则；依赖窄 store interface。
5. 在 `handlers` 复用 `DecodeJSON`、`Success`、`Error`、`ErrorCode`，保持响应形状。
6. 在 `server.go` 挂路由；复用现有 group 和 middleware。
7. 同步 `backend/internal/openapi/openapi.yaml` 与 `docs/openapi.yaml`。
8. 增加/更新邻近 Go tests，再运行对应包测试或 `cd backend && go test ./...`。

### chi 约定

Context7 查询的 `chi` 文档建议用 `Route` / `Group` / mounted subrouter 组织 REST API，并在 route group 上施加局部 middleware。本项目采用：

- 全局 middleware：`RequestIDMiddleware` -> `Recover` -> `CORS`；不要把 `SessionAuth` 挂成全局 middleware。
- Cookie 登录路由：`/auth`，其中 `GET /auth/me` 单独使用 `SessionAuth`。
- 已登录 portal API：`r.Group(... SessionAuth -> RequireUser ...)`。
- 管理 API：`/admin` + `SessionAuth` + `RequireUser` + `RequireAdmin`。
- Public API：`/v1` + `APIPreAuthRateLimit` + `APITokenAuth` + `APIRateLimit` + `APILastUsed` + `APIRequestLog`。

- `/v1` Redis 限流必须同时按 token、user、application 维度独立计数；不要只取 `Effective*Limit` 后按 token key 计数，否则多 token 会放大账号上限。
- Redis client pool/timeout 配置来自 `REDIS_POOL_SIZE`、`REDIS_MIN_IDLE_CONNS`、`REDIS_DIAL_TIMEOUT`、`REDIS_READ_TIMEOUT`、`REDIS_WRITE_TIMEOUT`、`REDIS_POOL_TIMEOUT`；timeout 值为 `0` 时由本项目转换为固定安全默认值（5s/3s/3s/4s），不要依赖 go-redis 版本默认语义；高 QPS `/v1` 调优应改配置，不要在 middleware 内创建额外 Redis client。
- `/v1` request logging 只允许通过 `services/requestlog.Writer` 的有界队列批量写入；不要在 middleware 中为每个请求启动 goroutine 或同步写 DB。Dashboard 统计应读取 `api_usage_*` 聚合表，raw `api_request_logs` 只短期保留用于排查，聚合明细按 dashboard 最大查询窗口清理。
- `ClientIP` 只在直接 peer 是 loopback/private/link-local 时信任 `X-Forwarded-For` / `X-Real-IP`，避免公网直连伪造 pre-auth IP 限流 key。

- Portal session 认证使用 Redis 短 TTL cache；`sessions.last_seen_at` 只能通过节流后的低频写入更新，避免每个带 cookie 的请求都触发 DB join + write。
改路由时优先把新 endpoint 放入现有职责 group；不要创建并行认证路径。

### pgx 约定

Context7 查询的 `pgx/v5` 文档要点：

- `pgxpool.Pool` 是并发安全连接池；repository 只依赖 `Queryer`。
- `QueryRow` 的错误延迟到 `Scan`；没行时处理 `pgx.ErrNoRows` 并映射到 `model.ErrNotFound` 或合适 sentinel error。
- `Query` 返回的 `Rows` 必须 `defer rows.Close()`，循环后检查 `rows.Err()`。
- 从 pool `Begin(ctx)` 后，必须 `Commit` 或 `Rollback`；`defer tx.Rollback(ctx)` 在成功 `Commit` 后是安全模式。
- `Exec` 返回 command tag；只有业务确实需要确认行数时才检查 `RowsAffected()`。
- PostgreSQL pool/timeout 配置来自 `DB_*`、`SYNC_DB_*`、`SOURCE_DB_*`；API 运行时使用 `DB_*` pool，sync 使用独立 `SYNC_DB_*` target pool 与 `SOURCE_DB_*` source pool。新增 DB 入口或后台任务时不要绕过 `repository.WithQueryTimeout`，不要重新让 API 与 full sync 共用同一个 target pool，也不要把大批量 sync 合并回单个长事务。

### 错误与响应

- 服务层返回 `model` sentinel errors；HTTP status/code 映射集中在 `handlers/respond.go`。
- `DecodeJSON` 使用 `DisallowUnknownFields()`；新增 handler 必须保持未知字段拒绝行为。
- `DecodeJSON` 会用 `http.MaxBytesReader` 限制请求体并拒绝额外 JSON/trailing bytes；新增 JSON handler 必须按 endpoint 选择 `smallJSONBodyLimit` 或 `applicationJSONBodyLimit`，不要直接读取无界 `r.Body`。
- 成功响应固定为 `{ "success": true, "data": ... }`。
- 失败响应固定为 `{ "success": false, "error": { "code", "message" } }`。
- 不把 DB 结构、SQL 错误、token、OTP、DSN、pepper 输出给客户端或日志。

## 同步改动流程

同步边界比 API 更严格：

- `SOURCE_DATABASE_DSN` 必须只读；本项目禁止修改主站数据库。
- 来源读取集中在 `backend/internal/services/sync/source_queries.go`，必须按 keyset/page 或游标分批读取；incremental 条件避免重新引入 `updated >= ... OR resource_update_time >= ...` 这类可能退化的大扫描。
- 字段清洗与默认值在 `mapper.go`。
- clean DB upsert、关系替换、search_text 刷新与 full-sync unseen deletion 在 repository/service 层；批量写入使用短事务 batch commit。
- full sync 通过 `sync_run_seen` staging 表记录当前 run 见过的 source patch，只有所有批次成功后才标记未见行 deleted；公开 API 默认只查 `deleted_at IS NULL` 且 SFW。

- Sync run 必须先拿到 PostgreSQL advisory lock 再创建 `sync_runs`；锁冲突直接返回 `model.ErrSyncRunning`，不要额外创建 failed run 记录。Kubernetes CronJob 部署必须保留 `concurrencyPolicy: Forbid`，让数据库锁只作为跨入口兜底，而不是常态重叠控制。

修改同步字段时需要同时检查：

1. 来源 SQL 是否只读取允许公开的字段。
2. clean DB migration 是否没有引入敏感数据列。
3. mapper 是否 trim/dedupe/default。
4. public repository/API 是否不返回 `source_*` 内部字段。
5. OpenAPI 与测试是否覆盖新增公开字段。

## 前端改动流程

Context7 查询的 Nuxt 3 文档要点：

- runtime config 在 `nuxt.config.ts` 的 `runtimeConfig` 中定义；客户端可见值必须放在 `runtimeConfig.public`。
- route middleware 使用 `defineNuxtRouteMiddleware((to, from) => ...)` 与 `navigateTo()`。
- 数据获取可以用 `$fetch` / `useFetch` / `useAsyncData`；本项目的后端调用统一封装在 `useApi()`。

本项目约定：

- 所有后端请求走 `frontend/composables/useApi.ts` 或基于它的 typed composables。
- `useApi()` 已设置 `credentials: 'include'`；SSR 时会转发请求 Cookie。
- 认证状态在 Pinia `frontend/stores/auth.ts`；页面保护在 `frontend/middleware/auth.ts` 与 `frontend/middleware/admin.ts`。
- 不在组件里硬编码后端地址；使用 `runtimeConfig.public.apiBaseUrl`。
- Nuxt/UI 改动后运行 `cd frontend && pnpm typecheck`；涉及 SSR、routes、config 时再运行 `pnpm build`。

## OpenAPI 与文档同步

API schema 或路由变更时，必须同时更新：

- `backend/internal/openapi/openapi.yaml`：后端 `/openapi.yaml` 服务的嵌入版本。
- `docs/openapi.yaml`：文档副本。

两份文件应保持内容一致。公共 API 的响应 schema 不得包含 `source_patch_id`、用户资料、下载链接、评论、上传者或内部权限字段。

## 测试与验证

- 后端行为改动：在相邻 package 增加/更新 `_test.go`，优先测服务层的规范化、状态机、权限、边界值与 sentinel error。
- middleware 改动：用 `httptest` 覆盖 header/cookie/context/rate-limit 分支。
- repository 改动：优先保持 SQL 小而直接；需要事务时用 `Queryer`/`NewWithQueryer` 注入。
- 前端改动：运行 `cd frontend && pnpm typecheck`；修改 Nuxt config、route structure 或 SSR-sensitive code 时运行 `pnpm build`。
- 全量检查：`make test` 等价于 `cd backend && go test ./...` 加 `cd frontend && pnpm typecheck`。


## 性能验证流程

- Go 基准：`make bench`；真实 Redis 限流压测需显式设置 `REDIS_BENCH_ADDR`，否则自动跳过。
- clean DB EXPLAIN：`make perf-explain DATABASE_DSN=...`，脚本在只读事务中覆盖 search、detail、dashboard 聚合查询。
- source DB EXPLAIN：`make perf-explain-source SOURCE_DATABASE_DSN=...`，必须使用主库只读账号，覆盖 full/incremental sync page query。
- Nuxt bundle：`make frontend-analyze`，底层运行 `nuxt analyze --no-serve`；不要提交 `.nuxt` 产物。
- Runtime 诊断：`ENABLE_PPROF` / `ENABLE_METRICS` 默认关闭；`ENABLE_PPROF=true` 时 `OBSERVABILITY_ADDR` 只能绑定 localhost/loopback；仅启用 metrics 时可绑定 private 管理地址；这些端点暴露 `/debug/pprof/*`、`/debug/pprof/trace`、`/debug/vars`，不要放到公网入口。

## 安全检查清单

交付前逐项确认：

- [ ] 是否仍只暴露 clean DB 中的公开元数据。
- [ ] 是否没有把主站内部 id 或敏感主站表字段加入公开响应。
- [ ] 是否没有记录 plaintext API token、OTP、session token、DSN、pepper。
- [ ] 发码 start endpoints 是否未绕过 Turnstile 校验，且未记录 Turnstile token。
- [ ] 是否保持 Cookie 登录而非前端存储 session。
- [ ] 是否保持 token hash + pepper 校验。
- [ ] 是否保持管理员路由 `RequireUser` + `RequireAdmin`。
- [ ] 是否保持 `/v1` token auth、request log、token/user/application 三维 rate limit middleware。
- [ ] 是否同步 OpenAPI 双份文件。
- [ ] 是否运行了直接覆盖改动的测试/类型检查。

- 邮件验证码驱动安全边界：`MAIL_DRIVER=log` 只能用于 `APP_ENV=development` 的本地调试；非开发环境必须使用真实 SMTP 或 Postal 配置，生产配置缺失应启动失败而不是回退日志驱动。
