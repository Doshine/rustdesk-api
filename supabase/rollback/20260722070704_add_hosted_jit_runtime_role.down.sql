begin;

revoke yinhe_app_runtime from yinhe_app_jit_runtime;
drop role if exists yinhe_app_jit_runtime;

commit;
