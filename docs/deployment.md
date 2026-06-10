# Deployment

1. 创建 `backend/.env` 与 `frontend/.env`。
2. 确认主机 PostgreSQL、Redis 已启动，并在 `backend/.env` 配置 `DATABASE_DSN`、`REDIS_ADDR`。
   按实际 QPS 调整 `DB_*`、`SYNC_DB_*`、`SOURCE_DB_*` 连接池与 timeout，保持 sync 使用独立 target/source pool；同时调整 `API_REQUEST_LOG_QUEUE_SIZE`、`API_REQUEST_LOG_BATCH_SIZE`、`API_REQUEST_LOG_FLUSH_INTERVAL`、`API_REQUEST_LOG_RETENTION_DAYS`，避免 raw request log 无界增长。
3. 如果后端运行在 Docker Compose 容器内，连接主机服务时将连接主机名配置为 `host.docker.internal`。
4. 只在独立 sync worker 的 env 中配置主库只读账号 `SOURCE_DATABASE_DSN`；API env 默认不需要 source DB 凭据。
   生产环境默认用 `backend/cmd/sync`、systemd timer 或 Kubernetes CronJob 独立跑同步，并保持 API `ENABLE_SYNC_WORKER=false`。Kubernetes CronJob 必须配置 `concurrencyPolicy: Forbid`；服务层仍使用 PostgreSQL advisory lock 兜底，防止 API 手动触发、API 内置 scheduler 与独立 worker 跨进程并发写 clean DB。仅本地调试或小数据量部署可启用 API 进程内 `ENABLE_SYNC_WORKER=true`，此时才把 `SOURCE_DATABASE_DSN` 配给 API 进程。
5. 执行 `make migrate-up`。
6. 执行 `make sync-full` 初始化 clean DB。
7. 启动 `docker compose -f deploy/docker-compose.yml up --build`。
