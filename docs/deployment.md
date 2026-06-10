# Deployment

1. 创建 `backend/.env` 与 `frontend/.env`。
2. 确认主机 PostgreSQL、Redis 已启动，并在 `backend/.env` 配置 `DATABASE_DSN`、`REDIS_ADDR`。
   按实际 QPS 调整 `API_REQUEST_LOG_QUEUE_SIZE`、`API_REQUEST_LOG_BATCH_SIZE`、`API_REQUEST_LOG_FLUSH_INTERVAL`、`API_REQUEST_LOG_RETENTION_DAYS`，避免 raw request log 无界增长。
3. 如果后端运行在 Docker Compose 容器内，连接主机服务时将连接主机名配置为 `host.docker.internal`。
4. 使用主库只读账号配置 `SOURCE_DATABASE_DSN`。
5. 执行 `make migrate-up`。
6. 执行 `make sync-full` 初始化 clean DB。
7. 启动 `docker compose -f deploy/docker-compose.yml up --build`。
