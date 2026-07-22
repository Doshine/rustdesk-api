\set ON_ERROR_STOP on

do $verify$
begin
    if (select max(version) from yinhe_app.versions) <> 273 then
        raise exception 'application database version is not 273';
    end if;
    if to_regclass('yinhe_app.deployment_codes') is null
        or to_regclass('yinhe_app.deployment_audit_events') is null then
        raise exception 'deployment enrollment tables are missing';
    end if;
    if not exists (
        select 1 from pg_indexes
        where schemaname = 'yinhe_app' and indexname = 'idx_deployment_codes_code_hash'
    ) then
        raise exception 'deployment code hash index is missing';
    end if;
    if not has_table_privilege('yinhe_app_runtime', 'yinhe_app.deployment_codes', 'UPDATE')
        or not has_table_privilege('yinhe_app_runtime', 'yinhe_app.deployment_audit_events', 'SELECT')
        or not has_table_privilege('yinhe_app_runtime', 'yinhe_app.deployment_audit_events', 'INSERT')
        or has_table_privilege('yinhe_app_runtime', 'yinhe_app.deployment_audit_events', 'UPDATE')
        or has_table_privilege('yinhe_app_runtime', 'yinhe_app.deployment_audit_events', 'DELETE') then
        raise exception 'deployment enrollment privileges are invalid';
    end if;
    if exists (
        select 1 from pg_class c
        join pg_namespace n on n.oid = c.relnamespace
        where n.nspname = 'yinhe_app'
          and c.relname in ('deployment_codes', 'deployment_audit_events')
          and not c.relrowsecurity
    ) then
        raise exception 'deployment enrollment RLS is not enabled';
    end if;
end
$verify$;

select jsonb_build_object(
    'status', 'PASS',
    'application_version', (select max(version) from yinhe_app.versions),
    'deployment_code_plaintext_column', exists (
        select 1 from information_schema.columns
        where table_schema = 'yinhe_app' and table_name = 'deployment_codes' and column_name = 'code'
    ),
    'audit_append_only', not has_table_privilege(
        'yinhe_app_runtime', 'yinhe_app.deployment_audit_events', 'UPDATE'
    )
) as verification;
