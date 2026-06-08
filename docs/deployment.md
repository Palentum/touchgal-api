# Deployment

1. 创建 `backend/.env` 与 `frontend/.env`。
2. 使用主库只读账号配置 `SOURCE_DATABASE_DSN`。
3. 执行 `make migrate-up`。
4. 执行 `make sync-full` 初始化 clean DB。
5. 启动 `docker compose -f deploy/docker-compose.yml up --build`。
