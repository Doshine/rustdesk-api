-- 自托管 PostgreSQL 兼容层（必须最先应用，文件名保证字典序最前）。
--
-- 背景：20260722050438_app_schema_272.sql 与 20260722095400_add_deployment_codes_273.sql
-- 中含有 `revoke ... from public, anon, authenticated, service_role;`。
-- anon / authenticated / service_role 是 Supabase 平台预置角色，自托管 PostgreSQL 上
-- 并不存在，REVOKE 会抛 `role "anon" does not exist`；由于整份迁移包在一个事务里，
-- 结果是整个 schema 迁移回滚——即"迁移文件在自托管环境下完全无法应用"。
--
-- 这里把这三个角色按需建成无登录、无任何权限的占位角色，使后续 revoke 语句
-- 在 Supabase 与自托管两种环境下逐字节保持一致，无需维护两套迁移。
-- 占位角色不被授予任何权限，且立刻被后续迁移显式 revoke；在 Supabase 上本文件为空操作。

begin;

do $compat$
declare
    r text;
    created text[] := '{}';
    bad text;
begin
    -- 迁移中实际引用到的全部 Supabase 内置角色。
    -- supabase_admin 也必须在内：272 基线第 1401 行用 pg_has_role(current_user,
    -- 'supabase_admin', 'MEMBER') 做守卫，而 pg_has_role 在角色不存在时**本身就报错**，
    -- 守卫并不能保护缺失角色的环境。
    foreach r in array array['anon', 'authenticated', 'service_role', 'supabase_admin'] loop
        if not exists (select 1 from pg_roles where rolname = r) then
            execute format(
                'create role %I nologin nosuperuser nocreatedb nocreaterole '
                || 'noinherit noreplication nobypassrls',
                r);
            created := created || r;
            execute format(
                'comment on role %I is %L',
                r,
                'Self-hosted placeholder for a Supabase built-in role; holds no privileges');
        end if;
    end loop;

    -- 断言只针对**本次新建**的占位角色。
    -- 不能对已存在的角色断言：真实 Supabase 环境里 supabase_admin 本就是超级用户，
    -- 一刀切的检查会让本文件在 Supabase 上直接抛异常。
    if array_length(created, 1) is not null then
        select string_agg(rolname, ', ')
          into bad
          from pg_roles
         where rolname = any(created)
           and (rolsuper or rolcreatedb or rolcreaterole
                or rolreplication or rolbypassrls or rolcanlogin);
        if bad is not null then
            raise exception 'compat placeholder roles unexpectedly hold elevated attributes: %', bad;
        end if;
    end if;

    raise notice 'selfhosted role compat: created [%]',
        coalesce(array_to_string(created, ', '), 'none — 已是 Supabase 环境');
end
$compat$;

commit;
