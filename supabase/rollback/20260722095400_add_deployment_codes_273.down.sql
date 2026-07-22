-- Roll back version 273 only when no enrollment history would be destroyed.
begin;

do $guard$
begin
    if exists (select 1 from yinhe_app.deployment_codes limit 1)
        or exists (select 1 from yinhe_app.deployment_audit_events limit 1) then
        raise exception 'rollback refused: deployment code or audit data exists';
    end if;
end
$guard$;

DELETE FROM yinhe_app.versions WHERE version = 273;
DROP TABLE yinhe_app.deployment_audit_events;
DROP TABLE yinhe_app.deployment_codes;

commit;
