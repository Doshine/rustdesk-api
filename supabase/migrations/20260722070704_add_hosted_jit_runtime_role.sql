begin;

do $$
begin
    if not exists (select 1 from pg_roles where rolname = 'yinhe_app_jit_runtime') then
        create role yinhe_app_jit_runtime
            login
            inherit
            nocreatedb
            nocreaterole
            password null;
    end if;
end
$$;

alter role yinhe_app_jit_runtime with
    login
    inherit
    nocreatedb
    nocreaterole
    password null;

do $$
begin
    if exists (
        select 1
        from pg_roles
        where rolname = 'yinhe_app_jit_runtime'
          and (rolsuper or rolcreatedb or rolcreaterole or rolreplication or rolbypassrls)
    ) then
        raise exception 'yinhe_app_jit_runtime unexpectedly has elevated role attributes';
    end if;
end
$$;

grant yinhe_app_runtime to yinhe_app_jit_runtime;

comment on role yinhe_app_jit_runtime is
    'Passwordless login role for time-limited Supabase JIT access; inherits only yinhe_app_runtime privileges';

commit;
