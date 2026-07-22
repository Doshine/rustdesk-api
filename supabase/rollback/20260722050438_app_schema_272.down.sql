-- Destructive rollback for the initial empty-database schema only.
-- Refuses to run when any business table contains data.
begin;

do $guard$
declare
    relation record;
    row_count bigint;
begin
    for relation in
        select tablename
        from pg_tables
        where schemaname = 'yinhe_app'
          and tablename <> 'versions'
        order by tablename
    loop
        execute format('select count(*) from yinhe_app.%I', relation.tablename) into row_count;
        if row_count <> 0 then
            raise exception 'rollback refused: yinhe_app.% contains % rows', relation.tablename, row_count;
        end if;
    end loop;
end
$guard$;

alter default privileges for role postgres in schema yinhe_app
    revoke all on tables from yinhe_app_runtime;
alter default privileges for role postgres in schema yinhe_app
    revoke all on sequences from yinhe_app_runtime;

drop schema yinhe_app cascade;
drop role if exists yinhe_app_runtime;

commit;
