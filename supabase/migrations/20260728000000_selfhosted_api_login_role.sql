-- 自托管部署用的应用登录角色。
--
-- 20260722050438 建立的 yinhe_app_runtime 是 NOLOGIN 权限角色，无法直接用于连接；
-- 20260722070704 建立的 yinhe_app_jit_runtime 是 Supabase JIT 专用（password null）。
-- 自托管环境需要一个可用口令登录、且只继承 yinhe_app_runtime 权限的角色。
--
-- 口令**不在本文件中设置**：由部署侧 init 脚本从挂载的密钥文件读入并经 stdin 执行
-- ALTER ROLE，避免口令进入迁移文件、镜像层、进程 argv 或版本库。

begin;

do $$
begin
    if not exists (select 1 from pg_roles where rolname = 'yinhe_app_api') then
        create role yinhe_app_api
            login
            inherit
            nosuperuser
            nocreatedb
            nocreaterole
            noreplication
            nobypassrls
            password null;
    end if;
end
$$;

alter role yinhe_app_api with
    login
    inherit
    nosuperuser
    nocreatedb
    nocreaterole
    noreplication
    nobypassrls;

-- 只继承 runtime 权限：普通业务表可 DML，审计表只允许 SELECT/INSERT，
-- versions 只读，schema 内 DDL 一律被拒绝（见 272 基线的授权部分）。
grant yinhe_app_runtime to yinhe_app_api;

-- 不给 public schema 的建表权，避免应用角色在默认 schema 里落对象
revoke create on schema public from yinhe_app_api;

do $$
begin
    if exists (
        select 1
        from pg_roles
        where rolname = 'yinhe_app_api'
          and (rolsuper or rolcreatedb or rolcreaterole or rolreplication or rolbypassrls)
    ) then
        raise exception 'yinhe_app_api unexpectedly has elevated role attributes';
    end if;
end
$$;

comment on role yinhe_app_api is
    'Least-privilege login role for the self-hosted API; inherits only yinhe_app_runtime';

commit;
