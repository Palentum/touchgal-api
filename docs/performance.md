# 性能验证文档

本文用于建立和复查 TouchGal API 的性能基线。所有 SQL 基线脚本都在 `BEGIN READ ONLY` 事务内运行并以 `ROLLBACK` 结束；涉及 source DB 的命令必须使用主站只读账号。

## 1. 验证目标

性能验证覆盖四类风险：

1. Go 服务热点：公开 API 参数处理、搜索 pattern 构建、sync batch、rate-limit 汇总逻辑。
2. clean DB 查询：公开搜索、搜索计数、游戏详情、标签/公司关系、dashboard 聚合。
3. source DB 查询：full sync keyset page、incremental changed page。
4. 前端 bundle：Nuxt 生产构建产物体积与依赖组成。

验证原则：

- 不对生产数据库执行写操作。
- 不在流量高峰对主库运行 `EXPLAIN (ANALYZE, BUFFERS)`。
- 不把 token、OTP、DSN、pepper 或用户请求原文写入日志和报告。
- 基线报告记录命令、参数、数据规模、关键耗时、buffer 命中/读取情况和异常查询计划。

## 2. 准备环境

本地或 CI 需要：

- Go `1.26.4`。
- PostgreSQL `psql` CLI。
- Node.js `22`、Corepack、`pnpm@11.5.0`。
- 可选 Redis，用于真实 Redis 限流 benchmark。

确认依赖：

```bash
go version
psql --version
node --version
pnpm --version
```

性能命令使用 root `Makefile`：

| 命令 | 作用 |
| --- | --- |
| `make bench` | 运行 Go benchmark。 |
| `make perf-explain` | 对 clean DB 公开查询执行 EXPLAIN。 |
| `make perf-explain-source` | 对 source DB 同步读取执行 EXPLAIN。 |
| `make frontend-analyze` | 运行 Nuxt bundle analysis。 |

## 3. Go benchmark

运行全部 Go benchmark：

```bash
make bench
```

等价于：

```bash
cd backend && go test -run='^$' -bench=. -benchmem ./...
```

默认行为：

- 覆盖项目中已有 benchmark。
- 输出每个 benchmark 的 ns/op、B/op、allocs/op。
- 真实 Redis benchmark 默认跳过，避免误打生产 Redis。

记录基线时保存：

- Git 提交/镜像标签或发布版本。
- CPU 型号与运行环境。
- Go 版本。
- benchmark 输出。
- 与上一基线相比的 ns/op、B/op、allocs/op 变化。

## 4. Redis 限流 benchmark

真实 Redis 限流 benchmark 只有显式设置 Redis 变量时才运行：

```bash
REDIS_BENCH_ADDR=localhost:6379 REDIS_BENCH_DB=15 make bench
```

可选密码：

```bash
REDIS_BENCH_ADDR=localhost:6379 REDIS_BENCH_PASSWORD='redis-password' REDIS_BENCH_DB=15 make bench
```

要求：

- 不要使用生产 Redis 主库。
- `REDIS_BENCH_DB` 指向隔离 logical DB。
- 观察 Redis CPU、内存、连接数和慢日志，确认 benchmark 未影响共享环境。

变量说明：

| 变量 | 必填 | 说明 |
| --- | --- | --- |
| `REDIS_BENCH_ADDR` | 是 | Redis benchmark 地址；为空时跳过真实 Redis benchmark。 |
| `REDIS_BENCH_PASSWORD` | 否 | Redis benchmark 密码。 |
| `REDIS_BENCH_DB` | 是 | Redis benchmark 隔离 DB；为空时跳过真实 Redis benchmark。 |

## 5. clean DB EXPLAIN 基线

运行默认 clean DB 查询基线：

```bash
DATABASE_DSN='postgres://touchgal_api:password@localhost:5432/touchgal_api?sslmode=disable' make perf-explain
```

默认参数：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `keyword` | `summer` | 搜索关键词。 |
| `page` | `1` | 搜索分页页码。 |
| `limit` | `20` | 搜索分页大小。 |
| `days` | `30` | dashboard 聚合窗口天数。 |
| `unique_id` | 空 | 游戏详情目标。为空时脚本自动选择第一条 SFW 未删除游戏。 |
| `user_id` | 空 | dashboard 目标用户。为空时脚本自动选择最近有聚合数据的用户。 |

指定参数示例：

```bash
DATABASE_DSN='postgres://touchgal_api:password@localhost:5432/touchgal_api?sslmode=disable' \
keyword='summer' page=1 limit=20 days=30 unique_id='game-unique-id' user_id='00000000-0000-0000-0000-000000000000' \
make perf-explain
```

脚本文件：`scripts/perf-explain.sql`。

脚本覆盖：

1. `clean-db search page query`：`games` SFW 未删除搜索分页。
2. `clean-db search count query`：同条件搜索计数。
3. `clean-db game detail primary query`：游戏详情主表与评分统计。
4. `clean-db game detail relation queries`：游戏标签与公司关系。
5. `clean-db dashboard aggregate query`：`api_usage_*` dashboard 聚合。

执行特性：

- `BEGIN READ ONLY`。
- `SET LOCAL statement_timeout = '30s'`。
- 使用 `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)`。
- `ROLLBACK` 结束。

观察重点：

- `Execution Time` 是否符合当前数据规模预期。
- `Buffers: shared hit/read/dirtied/written` 中是否出现异常 read 或写入。
- 搜索分页是否随 `page` 增大出现明显 offset 退化。
- dashboard 聚合是否读取聚合表，而不是 raw `api_request_logs`。
- 详情关系查询是否通过 join 表索引定位目标游戏。

## 6. source DB EXPLAIN 基线

运行 source DB 同步读取基线：

```bash
SOURCE_DATABASE_DSN='postgres://readonly_user:password@main-db-host:5432/touchgal?sslmode=require' make perf-explain-source
```

默认参数：

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| `source_last_id` | `0` | source `patch.id` keyset 起点。 |
| `source_limit` | `1000` | 每页读取数量。 |
| `source_window` | `1 day` | incremental changed page 的更新时间回看窗口。 |

指定参数示例：

```bash
SOURCE_DATABASE_DSN='postgres://readonly_user:password@main-db-host:5432/touchgal?sslmode=require' \
source_last_id=100000 source_limit=1000 source_window='6 hours' \
make perf-explain-source
```

脚本文件：`scripts/perf-explain-source.sql`。

脚本覆盖：

1. `source-db full sync keyset page query`：按 `patch.id > source_last_id` 顺序读取 full sync page。
2. `source-db incremental sync changed page query`：分别按 `updated` 与 `resource_update_time` 查找变更 id，再回表读取 page。

执行特性：

- `BEGIN READ ONLY`。
- `SET LOCAL statement_timeout = '30s'`。
- 使用 `EXPLAIN (ANALYZE, BUFFERS, FORMAT TEXT)`。
- `ROLLBACK` 结束。

安全要求：

- 只能使用主站只读账号。
- 不在主库写高峰运行。
- 发现顺序扫描或高 buffer read 后，先记录查询计划和数据规模，不要直接修改 source DB 索引；source DB schema 变更必须走主站数据库维护流程。

## 7. Nuxt bundle analysis

运行前端 bundle 分析：

```bash
make frontend-analyze
```

等价于：

```bash
cd frontend && pnpm analyze
```

底层命令是：

```bash
nuxt analyze --no-serve
```

记录内容：

- 最大 client chunk。
- 最大 server chunk。
- 新增大型依赖。
- ECharts、Nuxt UI、Vue 相关 chunk 是否符合预期。
- 与上一基线相比的主要变化。

要求：

- 不提交 `.nuxt`、`.output` 或分析产物。
- route、layout、Nuxt config 改动后，除了 analyze，还应运行 `cd frontend && pnpm typecheck`；SSR-sensitive 改动再运行 `cd frontend && pnpm build`。

## 8. Runtime diagnostics

默认关闭：

```env
ENABLE_PPROF=false
ENABLE_METRICS=false
OBSERVABILITY_ADDR=127.0.0.1:6060
```

开启 metrics：

```env
ENABLE_METRICS=true
OBSERVABILITY_ADDR=127.0.0.1:6060
```

开启 pprof：

```env
ENABLE_PPROF=true
OBSERVABILITY_ADDR=127.0.0.1:6060
```

端点：

| 路径 | 条件 | 说明 |
| --- | --- | --- |
| `/debug/vars` | `ENABLE_METRICS=true` | expvar 指标，包含 Go runtime 默认指标和 `api_request_log_*` 队列指标。 |
| `/debug/pprof/` | `ENABLE_PPROF=true` | pprof index。 |
| `/debug/pprof/profile` | `ENABLE_PPROF=true` | CPU profile。 |
| `/debug/pprof/trace` | `ENABLE_PPROF=true` | Go runtime trace。 |

绑定要求：

- `OBSERVABILITY_ADDR` 必须是 `host:port`。
- `ENABLE_PPROF=true` 时只能绑定 localhost 或 loopback。
- 仅开启 metrics 时可以绑定 private 管理地址。
- 禁止绑定 `:6060`、`0.0.0.0` 或公网 IP。
- 生产访问通过 SSH tunnel、kubectl port-forward 或内网管理面完成。

## 9. 性能报告格式

每次建立基线时记录：

```text
版本：
日期：
环境：
数据规模：games=, tags=, companies=, api_usage_daily=
命令：
参数：
结果摘要：
异常计划：
结论：
```

Go benchmark 摘要应包含：

- 退化超过预期的 benchmark 名称。
- `ns/op` 变化。
- `B/op` 变化。
- `allocs/op` 变化。

DB EXPLAIN 摘要应包含：

- 查询名。
- `Planning Time`。
- `Execution Time`。
- 主要节点类型。
- buffer hit/read。
- 是否出现顺序扫描、外部排序或高行数过滤。

前端 bundle 摘要应包含：

- 最大 chunk 名称与大小。
- 新增依赖。
- SSR build 是否通过。

## 10. 常见处理策略

### Go benchmark allocation 增加

处理顺序：

1. 确认 benchmark 环境一致。
2. 找到新增 allocation 的具体函数。
3. 优先删除不必要的字符串拼接、slice copy、map 构造和 interface boxing。
4. 用相同 benchmark 复测。

### clean DB 搜索变慢

处理顺序：

1. 对同一 `keyword/page/limit` 复跑 `make perf-explain`。
2. 记录搜索 page query 与 count query 的计划。
3. 检查 `games.deleted_at`、`content_limit`、`search_text` 条件选择性。
4. 优先调整查询或索引；不要扩大公开 API 响应范围。

### source DB 同步读取变慢

处理顺序：

1. 用相同 `source_last_id/source_limit/source_window` 复跑 `make perf-explain-source`。
2. 比较 full keyset page 与 incremental changed page。
3. 如果 incremental 回看窗口过大，评估 `SYNC_INCREMENTAL_SAFETY_MINUTES` 是否异常。
4. 如果需要 source DB 索引或 schema 调整，走主站数据库维护流程。

### request log 队列积压

处理顺序：

1. 开启 `ENABLE_METRICS=true` 并读取 `/debug/vars`。
2. 查看 `api_request_log_*` 队列指标。
3. 调整 `API_REQUEST_LOG_QUEUE_SIZE`、`API_REQUEST_LOG_BATCH_SIZE`、`API_REQUEST_LOG_FLUSH_INTERVAL`。
4. 检查 clean DB 写入延迟和连接池等待。
5. 不要在 middleware 中同步写 raw request log。
