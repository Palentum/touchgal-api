# Deployment

1. 创建 `backend/.env` 与 `frontend/.env`；生产必须在后端配置 `TURNSTILE_SECRET_KEY`，并在前端配置 `NUXT_PUBLIC_TURNSTILE_SITE_KEY`。
2. 确认主机 PostgreSQL、Redis 已启动，并在 `backend/.env` 配置 `DATABASE_DSN`、`REDIS_ADDR`。
   按实际 QPS 调整 `DB_*`、`SYNC_DB_*`、`SOURCE_DB_*` 连接池与 timeout，保持 sync 使用独立 target/source pool；同时调整 `REDIS_POOL_SIZE`、`REDIS_MIN_IDLE_CONNS`、`REDIS_DIAL_TIMEOUT`、`REDIS_READ_TIMEOUT`、`REDIS_WRITE_TIMEOUT`、`REDIS_POOL_TIMEOUT`，避免高并发 `/v1` 请求在 Redis pool wait 或慢 Redis I/O 上堆积；同时调整 `HTTP_READ_HEADER_TIMEOUT`、`HTTP_READ_TIMEOUT`、`HTTP_WRITE_TIMEOUT`、`HTTP_IDLE_TIMEOUT`、`HTTP_MAX_HEADER_BYTES` 与 nginx `client_max_body_size`/body timeout，避免慢客户端或超大请求体占用连接、goroutine、内存；`HTTP_WRITE_TIMEOUT` 必须大于正数 `DB_QUERY_TIMEOUT` 加 `HTTP_READ_TIMEOUT`，让后端仍能返回数据库超时错误；并调整 `API_REQUEST_LOG_QUEUE_SIZE`、`API_REQUEST_LOG_BATCH_SIZE`、`API_REQUEST_LOG_FLUSH_INTERVAL`、`API_REQUEST_LOG_RETENTION_DAYS`，避免 raw request log 无界增长。
   可选诊断端点通过 `ENABLE_PPROF`、`ENABLE_METRICS` 与 `OBSERVABILITY_ADDR` 控制，默认关闭。启用时只允许绑定 localhost、loopback 或 private 管理地址，并通过 SSH tunnel、kubectl port-forward 或内网管理面访问 `/debug/pprof/*`、`/debug/pprof/trace`、`/debug/vars`；不要经公网 Ingress 暴露。
3. 如果后端运行在 Docker Compose 容器内，连接主机服务时将连接主机名配置为 `host.docker.internal`。
4. 只在独立 sync worker 的 env 中配置主库只读账号 `SOURCE_DATABASE_DSN`；API env 默认不需要 source DB 凭据。
   生产环境默认用 `backend/cmd/sync`、systemd timer 或 Kubernetes CronJob 独立跑同步，并保持 API `ENABLE_SYNC_WORKER=false`。Kubernetes CronJob 必须配置 `concurrencyPolicy: Forbid`；服务层仍使用 PostgreSQL advisory lock 兜底，防止 API 手动触发、API 内置 scheduler 与独立 worker 跨进程并发写 clean DB。仅本地调试或小数据量部署可启用 API 进程内 `ENABLE_SYNC_WORKER=true`，此时才把 `SOURCE_DATABASE_DSN` 配给 API 进程。
   API readiness 使用 `/v1/ready`，只检查 clean PostgreSQL 与 Redis，不连接 source DB；Compose 示例用该端点做 backend healthcheck，并让 frontend 等待 `service_healthy`。
   Compose、systemd 与 Kubernetes 示例都包含默认 CPU/memory/task 限制；生产应按实例规格和同步批量大小调整，而不是移除限制。
5. 执行 `make migrate-up`。
6. 执行 `make sync-full` 初始化 clean DB。
7. 启动 `docker compose -f deploy/docker-compose.yml up --build`。
   `deploy/nginx.conf` 是完整 nginx 示例：`/api/` 做请求/连接限流、超时与 `no-store`，全站设置 HSTS、CSP、`frame-ancestors`/`X-Frame-Options`、`Referrer-Policy`、`Permissions-Policy` 与 `X-Content-Type-Options`，`/_nuxt/` 使用 immutable 长缓存，`/logo.webp` 使用短 TTL，避免固定 public 文件名长期失效困难。若 `NUXT_PUBLIC_API_BASE_URL` 指向非同源 API，在 nginx CSP 的 `connect-src` 中同步加入该 API origin。
