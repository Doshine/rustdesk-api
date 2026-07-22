-- Phase 3 device enrollment: scoped deployment codes and append-only evidence.
begin;

CREATE TABLE yinhe_app.deployment_codes (
    id bigserial PRIMARY KEY,
    name text DEFAULT ''::text NOT NULL,
    code_hash varchar(64) NOT NULL,
    code_hint text DEFAULT ''::text NOT NULL,
    device_group_id bigint DEFAULT 0 NOT NULL,
    allowed_platforms text NOT NULL,
    expires_at bigint DEFAULT 0 NOT NULL,
    max_uses bigint DEFAULT 1 NOT NULL,
    used_count bigint DEFAULT 0 NOT NULL,
    status bigint DEFAULT 1 NOT NULL,
    created_by bigint DEFAULT 0 NOT NULL,
    revoked_by bigint DEFAULT 0 NOT NULL,
    revoked_at bigint DEFAULT 0 NOT NULL,
    rotated_from_id bigint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone,
    CONSTRAINT deployment_codes_capacity_check CHECK (max_uses BETWEEN 1 AND 10000),
    CONSTRAINT deployment_codes_used_count_check CHECK (used_count >= 0 AND used_count <= max_uses),
    CONSTRAINT deployment_codes_status_check CHECK (status IN (1, 2))
);

CREATE UNIQUE INDEX idx_deployment_codes_code_hash ON yinhe_app.deployment_codes (code_hash);
CREATE INDEX idx_deployment_codes_device_group_id ON yinhe_app.deployment_codes (device_group_id);
CREATE INDEX idx_deployment_codes_expires_at ON yinhe_app.deployment_codes (expires_at);
CREATE INDEX idx_deployment_codes_status ON yinhe_app.deployment_codes (status);
CREATE INDEX idx_deployment_codes_created_by ON yinhe_app.deployment_codes (created_by);
CREATE INDEX idx_deployment_codes_rotated_from_id ON yinhe_app.deployment_codes (rotated_from_id);

CREATE TABLE yinhe_app.deployment_audit_events (
    id bigserial PRIMARY KEY,
    action text DEFAULT ''::text NOT NULL,
    deployment_code_id bigint DEFAULT 0 NOT NULL,
    actor_user_id bigint DEFAULT 0 NOT NULL,
    peer_row_id bigint DEFAULT 0 NOT NULL,
    device_id text DEFAULT ''::text NOT NULL,
    uuid text DEFAULT ''::text NOT NULL,
    ip text DEFAULT ''::text NOT NULL,
    detail text DEFAULT ''::text NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);

CREATE INDEX idx_deployment_audit_events_action ON yinhe_app.deployment_audit_events (action);
CREATE INDEX idx_deployment_audit_events_deployment_code_id ON yinhe_app.deployment_audit_events (deployment_code_id);
CREATE INDEX idx_deployment_audit_events_actor_user_id ON yinhe_app.deployment_audit_events (actor_user_id);
CREATE INDEX idx_deployment_audit_events_peer_row_id ON yinhe_app.deployment_audit_events (peer_row_id);
CREATE INDEX idx_deployment_audit_events_device_id ON yinhe_app.deployment_audit_events (device_id);
CREATE INDEX idx_deployment_audit_events_uuid ON yinhe_app.deployment_audit_events (uuid);

ALTER TABLE yinhe_app.deployment_codes ENABLE ROW LEVEL SECURITY;
CREATE POLICY yinhe_app_runtime_all ON yinhe_app.deployment_codes
    FOR ALL TO yinhe_app_runtime USING (true) WITH CHECK (true);

ALTER TABLE yinhe_app.deployment_audit_events ENABLE ROW LEVEL SECURITY;
CREATE POLICY yinhe_app_runtime_select ON yinhe_app.deployment_audit_events
    FOR SELECT TO yinhe_app_runtime USING (true);
CREATE POLICY yinhe_app_runtime_insert ON yinhe_app.deployment_audit_events
    FOR INSERT TO yinhe_app_runtime WITH CHECK (true);

GRANT SELECT, INSERT, UPDATE, DELETE ON yinhe_app.deployment_codes TO yinhe_app_runtime;
GRANT SELECT, INSERT ON yinhe_app.deployment_audit_events TO yinhe_app_runtime;
-- The 272 baseline grants DML on future tables through default privileges.
-- Revoke mutation explicitly so enrollment evidence remains append-only.
REVOKE UPDATE, DELETE ON yinhe_app.deployment_audit_events FROM yinhe_app_runtime;
GRANT USAGE, SELECT ON SEQUENCE yinhe_app.deployment_codes_id_seq TO yinhe_app_runtime;
GRANT USAGE, SELECT ON SEQUENCE yinhe_app.deployment_audit_events_id_seq TO yinhe_app_runtime;

REVOKE ALL ON yinhe_app.deployment_codes, yinhe_app.deployment_audit_events
    FROM public, anon, authenticated, service_role;
REVOKE ALL ON SEQUENCE yinhe_app.deployment_codes_id_seq, yinhe_app.deployment_audit_events_id_seq
    FROM public, anon, authenticated, service_role;

INSERT INTO yinhe_app.versions (version, created_at, updated_at)
SELECT 273, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM yinhe_app.versions WHERE version = 273);

commit;
