# TouchGal Developer/API Architecture

主项目数据库只作为只读 source DB。同步任务读取 Galgame 条目元数据，写入本项目独立 PostgreSQL clean DB。公开 API 只查询 clean DB，并通过独立账号、独立申请、独立 API token 授权。

数据流：KunMoe 主库 -> sync worker -> sanitized PostgreSQL -> /v1 API -> Developer Portal。
