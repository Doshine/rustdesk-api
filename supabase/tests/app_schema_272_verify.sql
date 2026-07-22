\set ON_ERROR_STOP on

do $verify$
declare
    expected_tables text[] := array[
        'address_book_collection_rules', 'address_book_collections', 'address_books',
        'audit_conns', 'audit_files', 'device_groups', 'groups', 'login_logs',
        'oauths', 'passkey_credentials', 'peers', 'relay_nodes', 'server_cmds',
        'share_records', 'tags', 'user_thirds', 'user_tokens', 'users', 'versions'
    ];
    actual_tables text[];
    runtime_role record;
begin
    if not exists (select 1 from pg_namespace where nspname = 'yinhe_app') then
        raise exception 'yinhe_app schema is missing';
    end if;

    select array_agg(tablename order by tablename)
    into actual_tables
    from pg_tables
    where schemaname = 'yinhe_app';

    select array_agg(value order by value)
    into expected_tables
    from unnest(expected_tables) as value;

    if actual_tables is distinct from expected_tables then
        raise exception 'application table set mismatch: %', actual_tables;
    end if;

    if (select max(version) from yinhe_app.versions) <> 272 then
        raise exception 'application database version is not 272';
    end if;

    if exists (
        select 1 from pg_class c
        join pg_namespace n on n.oid = c.relnamespace
        where n.nspname = 'yinhe_app' and c.relkind in ('r', 'p') and not c.relrowsecurity
    ) then
        raise exception 'one or more application tables do not have RLS enabled';
    end if;

    select rolcanlogin, rolsuper, rolcreatedb, rolcreaterole, rolreplication, rolbypassrls
    into runtime_role
    from pg_roles
    where rolname = 'yinhe_app_runtime';
    if not found or runtime_role.rolcanlogin or runtime_role.rolsuper
        or runtime_role.rolcreatedb or runtime_role.rolcreaterole
        or runtime_role.rolreplication or runtime_role.rolbypassrls then
        raise exception 'yinhe_app_runtime role boundary is invalid';
    end if;

    if not has_schema_privilege('yinhe_app_runtime', 'yinhe_app', 'USAGE') then
        raise exception 'runtime role cannot use yinhe_app schema';
    end if;
    if has_schema_privilege('anon', 'yinhe_app', 'USAGE')
        or has_schema_privilege('authenticated', 'yinhe_app', 'USAGE')
        or has_schema_privilege('service_role', 'yinhe_app', 'USAGE') then
        raise exception 'a Supabase API role can use the private application schema';
    end if;

    if not has_table_privilege('yinhe_app_runtime', 'yinhe_app.users', 'SELECT')
        or not has_table_privilege('yinhe_app_runtime', 'yinhe_app.users', 'INSERT')
        or not has_table_privilege('yinhe_app_runtime', 'yinhe_app.users', 'UPDATE')
        or not has_table_privilege('yinhe_app_runtime', 'yinhe_app.users', 'DELETE') then
        raise exception 'runtime role is missing normal application DML privileges';
    end if;
    if not has_table_privilege('yinhe_app_runtime', 'yinhe_app.audit_conns', 'SELECT')
        or not has_table_privilege('yinhe_app_runtime', 'yinhe_app.audit_conns', 'INSERT')
        or has_table_privilege('yinhe_app_runtime', 'yinhe_app.audit_conns', 'UPDATE')
        or has_table_privilege('yinhe_app_runtime', 'yinhe_app.audit_conns', 'DELETE')
        or not has_table_privilege('yinhe_app_runtime', 'yinhe_app.audit_files', 'SELECT')
        or not has_table_privilege('yinhe_app_runtime', 'yinhe_app.audit_files', 'INSERT')
        or has_table_privilege('yinhe_app_runtime', 'yinhe_app.audit_files', 'UPDATE')
        or has_table_privilege('yinhe_app_runtime', 'yinhe_app.audit_files', 'DELETE') then
        raise exception 'append-only audit table privileges are invalid';
    end if;
    if not has_table_privilege('yinhe_app_runtime', 'yinhe_app.versions', 'SELECT')
        or has_table_privilege('yinhe_app_runtime', 'yinhe_app.versions', 'INSERT')
        or has_table_privilege('yinhe_app_runtime', 'yinhe_app.versions', 'UPDATE')
        or has_table_privilege('yinhe_app_runtime', 'yinhe_app.versions', 'DELETE') then
        raise exception 'runtime version-table privileges are invalid';
    end if;

    if not exists (
        select 1 from pg_indexes
        where schemaname = 'yinhe_app' and indexname = 'idx_passkey_credential_rp'
    ) or not exists (
        select 1 from pg_indexes
        where schemaname = 'yinhe_app' and indexname = 'idx_user_tokens_passkey_credential_id'
    ) or not exists (
        select 1 from pg_indexes
        where schemaname = 'yinhe_app' and indexname = 'idx_login_logs_passkey_credential_id'
    ) then
        raise exception 'required Passkey indexes are missing';
    end if;

    if exists (
        select 1
        from pg_tables t
        cross join unnest(array['anon', 'authenticated', 'service_role']) as roles(role_name)
        where t.schemaname = 'yinhe_app'
          and (
              has_table_privilege(role_name, format('%I.%I', t.schemaname, t.tablename), 'SELECT')
              or has_table_privilege(role_name, format('%I.%I', t.schemaname, t.tablename), 'INSERT')
              or has_table_privilege(role_name, format('%I.%I', t.schemaname, t.tablename), 'UPDATE')
              or has_table_privilege(role_name, format('%I.%I', t.schemaname, t.tablename), 'DELETE')
          )
    ) then
        raise exception 'a Supabase API role has application table privileges';
    end if;
end
$verify$;

select jsonb_build_object(
    'status', 'PASS',
    'schema', 'yinhe_app',
    'application_version', (select max(version) from yinhe_app.versions),
    'tables', (select count(*) from pg_tables where schemaname = 'yinhe_app'),
    'rls_tables', (
        select count(*) from pg_class c
        join pg_namespace n on n.oid = c.relnamespace
        where n.nspname = 'yinhe_app' and c.relkind in ('r', 'p') and c.relrowsecurity
    ),
    'policies', (select count(*) from pg_policies where schemaname = 'yinhe_app'),
    'runtime_role', 'yinhe_app_runtime'
) as verification;
