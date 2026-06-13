# TouchGal API Architecture

主项目数据库只作为只读 source DB。同步任务读取 Galgame 条目元数据，写入本项目独立 PostgreSQL clean DB。公开 API 只查询 clean DB，并通过独立账号、独立申请、独立 API token 授权。

数据流：TouchGal 主库 → sync worker → clean PostgreSQL → /v1 API → Nuxt Developer Portal。

`/v1` 请求日志进入 API 进程内有界队列，由固定 writer 批量写入 `api_request_logs` 与 `api_usage_*` 聚合表。dashboard 只读聚合表，raw log 通过保留策略短期清理，聚合表按 dashboard 最大查询窗口保留。
