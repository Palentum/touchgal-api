# Performance Verification

本流程只建立性能基线，不修改业务数据。生产排障优先在只读副本或低峰窗口执行；涉及 source DB 的命令必须使用只读账号。

## Go benchmark

```bash
make bench
```

该目标运行 `cd backend && go test -run='^$' -bench=. -benchmem ./...`，覆盖公开 API 参数规范化、搜索 pattern 构建、sync rating batch、rate-limit 汇总逻辑等热点。真实 Redis 限流 benchmark 默认跳过；需要压测 Redis 计数脚本时显式提供专用 Redis 或专用 DB：

```bash
REDIS_BENCH_ADDR=localhost:6379 REDIS_BENCH_DB=15 make bench
```

不要指向生产 Redis 主库；必须显式设置 `REDIS_BENCH_DB` 指向隔离 DB，benchmark 会使用唯一 token/user/application key，TTL 自动清理。

## DB EXPLAIN

clean DB 公开查询基线：

```bash
DATABASE_DSN='postgres://touchgal_api:...@localhost:5432/touchgal_api?sslmode=disable' make perf-explain
```

可选变量：

```bash
keyword=summer page=1 limit=20 days=30 unique_id=abcd1234 user_id=00000000-0000-0000-0000-000000000000 make perf-explain
```

source DB 同步读取基线：

```bash
SOURCE_DATABASE_DSN='postgres://readonly_user:...@main-db:5432/touchgal?sslmode=require' make perf-explain-source
```

可选变量：

```bash
source_last_id=0 source_limit=1000 source_window='1 day' make perf-explain-source
```

两个脚本都在 `BEGIN READ ONLY` 事务内运行并设置 30 秒 `statement_timeout`。脚本使用 `EXPLAIN (ANALYZE, BUFFERS)`，会执行 SELECT；不要在流量高峰或主库写压力高时运行。

慢查询排查优先开启 PostgreSQL 侧 `pg_stat_statements` 或 `log_min_duration_statement`，记录语句形状和耗时即可；不要把 token、OTP、DSN、pepper 或请求参数原文写入应用日志。

## Nuxt bundle analysis

```bash
make frontend-analyze
```

该目标运行 `cd frontend && pnpm analyze`，底层是 Nuxt 3 的 `nuxt analyze --no-serve`。提交前只需保留报告结论，不要提交 `.nuxt` 产物。

## Runtime diagnostics

默认关闭：

```env
ENABLE_PPROF=false
ENABLE_METRICS=false
OBSERVABILITY_ADDR=127.0.0.1:6060
```

启用后 API 进程会额外启动只读诊断 HTTP server：

- `/debug/pprof/`、`/debug/pprof/profile`：CPU/heap/goroutine pprof。
- `/debug/pprof/trace`：Go runtime trace。
- `/debug/vars`：expvar 指标，包含 Go runtime 默认指标和 `api_request_log_*` 队列指标。

`OBSERVABILITY_ADDR` 校验为 localhost、loopback 或 private 管理地址；不要绑定 `:6060`、`0.0.0.0` 或公网 IP。生产建议通过 SSH tunnel、kubectl port-forward 或内网管理面访问。
