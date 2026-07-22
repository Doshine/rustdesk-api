# 蓝鲸银河 Supabase PostgreSQL 迁移

本目录保存 `rustdesk-api` 的 PostgreSQL 显式迁移。当前基线由真实 GORM
`AutoMigrate` 在 PostgreSQL 17.10 中生成，再收敛为可审计的版本 272 SQL。

## 边界

- 业务对象只位于私有 `yinhe_app` schema，不加入 Supabase Data API exposed schemas。
- `anon`、`authenticated`、`service_role` 无 schema、表、序列或函数权限。
- `yinhe_app_runtime` 是 `NOLOGIN` 权限角色；`yinhe_app_jit_runtime` 是最小权限登录角色，只继承前者且迁移不设置静态密码。正式部署凭据必须只进入密钥管理系统，不写入仓库或迁移。
- 运行时角色可正常操作业务表，但审计表只能 `SELECT/INSERT`，`versions` 只能 `SELECT`。
- 应用优先使用 Direct Connection；免费项目 Direct endpoint 仅 IPv6，IPv4 持久后端使用 Session Pooler 5432。当前 GORM/pgx 路径禁止 Transaction Pooler 6543。
- `postgresql.jit-access` 默认关闭；仅在 Supabase Temporary Access token 可用时开启，并通过环境变量注入 token。不得把 token 写入 YAML、命令历史或日志。
- 生产启动必须设置 `gorm.auto-migrate=false`，由启动检查确认数据库版本严格等于应用版本。

## 操作顺序

1. 在隔离 PostgreSQL 17 上执行 `supabase/migrations/*_app_schema_272.sql`。
2. 使用 `supabase/tests/app_schema_272_verify.sql` 验证对象、版本、RLS 和权限。
3. 使用迁移创建的 `yinhe_app_jit_runtime` 或由部署系统创建同等最小权限登录角色；正式凭据只从密钥管理系统注入，不得写入 SQL、配置模板或日志。
4. 通过环境变量注入连接参数，并以 `postgresql.schema=yinhe_app`、`postgresql.pool-mode=direct|session`、`gorm.auto-migrate=false` 启动后端。
5. 仅在空库回滚演练中使用 `supabase/rollback/*_app_schema_272.down.sql`。脚本检测到业务数据会拒绝执行。

应用管理员账号和业务数据不属于 schema 迁移。它们必须通过独立、可审计的初始化或数据迁移流程创建。

## 托管连接证据

`lib/orm/postgresql_hosted_test.go` 是显式 opt-in 的托管烟测，只从环境变量读取短期秘密，并通过应用真实 DSN 构造和 GORM/pgx 路径验证：当前角色、私有 schema、PostgreSQL 版本、TLS、版本 272 与最小权限。测试完成后必须清空远端 verifier、关闭 Temporary Access、撤销临时 CLI token，并确认测试业务行归零。

2026-07-22 的 `yinhe` 测试项目已完成 Session Pooler + TLS 1.3 验证。Temporary Access/JIT 的 API 配置成功，但平台 token 在 Supavisor 返回 PAM authentication failed，因此该能力明确记录为未通过，不能作为部署方案依赖。Direct 连接因本机 IPv4 与免费项目 IPv6-only endpoint 未执行；MySQL 也不在本轮范围内。
