-- 蓝鲸银河 PostgreSQL 17 / GORM schema baseline, application version 272.
-- Generated from the current model set with real GORM AutoMigrate, then
-- hardened for a private Supabase schema and a non-login runtime role.
begin;

-- Dumped from database version 17.10
-- Dumped by pg_dump version 17.10

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: yinhe_app; Type: SCHEMA; Schema: -; Owner: -
--

CREATE SCHEMA IF NOT EXISTS yinhe_app AUTHORIZATION postgres;

do $role$
begin
    if not exists (select 1 from pg_roles where rolname = 'yinhe_app_runtime') then
        create role yinhe_app_runtime
            nologin nosuperuser nocreatedb nocreaterole noinherit noreplication nobypassrls;
    else
        alter role yinhe_app_runtime
            nologin nosuperuser nocreatedb nocreaterole noinherit noreplication nobypassrls;
    end if;
end
$role$;

revoke all on schema yinhe_app from public, anon, authenticated, service_role;
grant usage on schema yinhe_app to yinhe_app_runtime;

-- The application uses a direct PostgreSQL connection, not the Data API.
-- Keep future public objects opt-in even if project-level defaults change.
revoke create on schema public from public;
alter default privileges for role postgres in schema public
    revoke all on tables from anon, authenticated, service_role;
alter default privileges for role postgres in schema public
    revoke all on sequences from anon, authenticated, service_role;
alter default privileges for role postgres in schema public
    revoke all on functions from public, anon, authenticated, service_role;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: address_book_collection_rules; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.address_book_collection_rules (
    id bigint NOT NULL,
    user_id bigint DEFAULT 0 NOT NULL,
    collection_id bigint DEFAULT 0 NOT NULL,
    rule bigint DEFAULT 0 NOT NULL,
    type bigint DEFAULT 1 NOT NULL,
    to_id bigint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: address_book_collection_rules_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.address_book_collection_rules_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: address_book_collection_rules_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.address_book_collection_rules_id_seq OWNED BY yinhe_app.address_book_collection_rules.id;


--
-- Name: address_book_collections; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.address_book_collections (
    id bigint NOT NULL,
    user_id bigint DEFAULT 0 NOT NULL,
    name text DEFAULT ''::text NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: address_book_collections_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.address_book_collections_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: address_book_collections_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.address_book_collections_id_seq OWNED BY yinhe_app.address_book_collections.id;


--
-- Name: address_books; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.address_books (
    row_id bigint NOT NULL,
    id text DEFAULT '0'::text NOT NULL,
    username text DEFAULT ''::text NOT NULL,
    password text DEFAULT ''::text NOT NULL,
    hostname text DEFAULT ''::text NOT NULL,
    alias text DEFAULT ''::text NOT NULL,
    platform text DEFAULT ''::text NOT NULL,
    tags text NOT NULL,
    hash text DEFAULT ''::text NOT NULL,
    user_id bigint DEFAULT 0 NOT NULL,
    force_always_relay boolean DEFAULT false NOT NULL,
    rdp_port text DEFAULT ''::text NOT NULL,
    rdp_username text DEFAULT ''::text NOT NULL,
    online boolean DEFAULT false NOT NULL,
    login_name text DEFAULT ''::text NOT NULL,
    same_server boolean DEFAULT false NOT NULL,
    collection_id bigint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: address_books_row_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.address_books_row_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: address_books_row_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.address_books_row_id_seq OWNED BY yinhe_app.address_books.row_id;


--
-- Name: audit_conns; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.audit_conns (
    id bigint NOT NULL,
    action text DEFAULT ''::text NOT NULL,
    conn_id bigint DEFAULT 0 NOT NULL,
    peer_id text DEFAULT ''::text NOT NULL,
    from_peer text DEFAULT ''::text NOT NULL,
    from_name text DEFAULT ''::text NOT NULL,
    ip text DEFAULT ''::text NOT NULL,
    session_id text DEFAULT ''::text NOT NULL,
    type bigint DEFAULT 0 NOT NULL,
    uuid text DEFAULT ''::text NOT NULL,
    close_time bigint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: audit_conns_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.audit_conns_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: audit_conns_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.audit_conns_id_seq OWNED BY yinhe_app.audit_conns.id;


--
-- Name: audit_files; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.audit_files (
    id bigint NOT NULL,
    from_peer text DEFAULT ''::text NOT NULL,
    info text DEFAULT ''::text NOT NULL,
    is_file boolean DEFAULT false NOT NULL,
    path text DEFAULT ''::text NOT NULL,
    peer_id text DEFAULT ''::text NOT NULL,
    type bigint DEFAULT 0 NOT NULL,
    uuid text DEFAULT ''::text NOT NULL,
    ip text DEFAULT ''::text NOT NULL,
    num bigint DEFAULT 0 NOT NULL,
    from_name text DEFAULT ''::text NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: audit_files_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.audit_files_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: audit_files_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.audit_files_id_seq OWNED BY yinhe_app.audit_files.id;


--
-- Name: device_groups; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.device_groups (
    id bigint NOT NULL,
    name text DEFAULT ''::text NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: device_groups_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.device_groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: device_groups_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.device_groups_id_seq OWNED BY yinhe_app.device_groups.id;


--
-- Name: groups; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.groups (
    id bigint NOT NULL,
    name text DEFAULT ''::text NOT NULL,
    type bigint DEFAULT 1 NOT NULL,
    route_names text DEFAULT ''::text NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: groups_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: groups_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.groups_id_seq OWNED BY yinhe_app.groups.id;


--
-- Name: login_logs; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.login_logs (
    id bigint NOT NULL,
    user_id bigint DEFAULT 0 NOT NULL,
    client text,
    device_id text,
    uuid text,
    ip text,
    type text,
    platform text,
    user_token_id bigint DEFAULT 0 NOT NULL,
    passkey_credential_id bigint DEFAULT 0 NOT NULL,
    is_deleted bigint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: login_logs_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.login_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: login_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.login_logs_id_seq OWNED BY yinhe_app.login_logs.id;


--
-- Name: oauths; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.oauths (
    id bigint NOT NULL,
    op text,
    oauth_type text,
    client_id text,
    client_secret text,
    auto_register boolean,
    scopes text,
    issuer text,
    pkce_enable boolean,
    pkce_method text,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: oauths_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.oauths_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: oauths_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.oauths_id_seq OWNED BY yinhe_app.oauths.id;


--
-- Name: passkey_credentials; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.passkey_credentials (
    id bigint NOT NULL,
    user_id bigint DEFAULT 0 NOT NULL,
    rp_id text DEFAULT ''::text NOT NULL,
    credential_id character varying(512) DEFAULT ''::character varying NOT NULL,
    user_handle character varying(128) DEFAULT ''::character varying NOT NULL,
    name character varying(128) DEFAULT ''::character varying NOT NULL,
    credential_data text NOT NULL,
    flags_raw smallint DEFAULT 0 NOT NULL,
    sign_count bigint DEFAULT 0 NOT NULL,
    clone_warning boolean DEFAULT false NOT NULL,
    backup_eligible boolean DEFAULT false NOT NULL,
    backup_state boolean DEFAULT false NOT NULL,
    last_used_at bigint DEFAULT 0 NOT NULL,
    revoked_at bigint,
    revoked_by bigint DEFAULT 0 NOT NULL,
    revocation_reason character varying(255) DEFAULT ''::character varying NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: passkey_credentials_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.passkey_credentials_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: passkey_credentials_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.passkey_credentials_id_seq OWNED BY yinhe_app.passkey_credentials.id;


--
-- Name: peers; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.peers (
    row_id bigint NOT NULL,
    id text DEFAULT ''::text NOT NULL,
    cpu text DEFAULT ''::text NOT NULL,
    hostname text DEFAULT ''::text NOT NULL,
    memory text DEFAULT ''::text NOT NULL,
    os text DEFAULT ''::text NOT NULL,
    username text DEFAULT ''::text NOT NULL,
    uuid text DEFAULT ''::text NOT NULL,
    version text DEFAULT ''::text NOT NULL,
    user_id bigint DEFAULT 0 NOT NULL,
    last_online_time bigint DEFAULT 0 NOT NULL,
    last_online_ip text DEFAULT ''::text NOT NULL,
    group_id bigint DEFAULT 0 NOT NULL,
    alias text DEFAULT ''::text NOT NULL,
    status bigint DEFAULT 1 NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: peers_row_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.peers_row_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: peers_row_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.peers_row_id_seq OWNED BY yinhe_app.peers.row_id;


--
-- Name: relay_nodes; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.relay_nodes (
    row_id bigint NOT NULL,
    name text DEFAULT ''::text NOT NULL,
    host text DEFAULT ''::text NOT NULL,
    port bigint DEFAULT 21117 NOT NULL,
    public_key text DEFAULT ''::text NOT NULL,
    region text DEFAULT ''::text NOT NULL,
    isp text DEFAULT ''::text NOT NULL,
    priority bigint DEFAULT 0 NOT NULL,
    enabled boolean NOT NULL,
    status bigint DEFAULT 0 NOT NULL,
    last_latency_ms bigint DEFAULT 0 NOT NULL,
    last_checked_at bigint DEFAULT 0 NOT NULL,
    remark text DEFAULT ''::text NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: relay_nodes_row_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.relay_nodes_row_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: relay_nodes_row_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.relay_nodes_row_id_seq OWNED BY yinhe_app.relay_nodes.row_id;


--
-- Name: server_cmds; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.server_cmds (
    id bigint NOT NULL,
    cmd text DEFAULT ''::text NOT NULL,
    alias text DEFAULT ''::text NOT NULL,
    option text DEFAULT ''::text NOT NULL,
    explain text DEFAULT ''::text NOT NULL,
    target text DEFAULT ''::text NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: server_cmds_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.server_cmds_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: server_cmds_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.server_cmds_id_seq OWNED BY yinhe_app.server_cmds.id;


--
-- Name: share_records; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.share_records (
    id bigint NOT NULL,
    user_id bigint DEFAULT 0 NOT NULL,
    peer_id text DEFAULT ''::text NOT NULL,
    share_token text DEFAULT ''::text NOT NULL,
    password_type text DEFAULT ''::text NOT NULL,
    password text DEFAULT ''::text NOT NULL,
    expire bigint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: share_records_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.share_records_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: share_records_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.share_records_id_seq OWNED BY yinhe_app.share_records.id;


--
-- Name: tags; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.tags (
    id bigint NOT NULL,
    name text DEFAULT ''::text NOT NULL,
    user_id bigint DEFAULT 0 NOT NULL,
    color bigint DEFAULT 0 NOT NULL,
    collection_id bigint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: tags_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.tags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: tags_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.tags_id_seq OWNED BY yinhe_app.tags.id;


--
-- Name: user_thirds; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.user_thirds (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    open_id text NOT NULL,
    name text,
    username text,
    email text,
    verified_email boolean,
    picture text,
    union_id text DEFAULT ''::text NOT NULL,
    third_type text DEFAULT ''::text NOT NULL,
    oauth_type text DEFAULT ''::text NOT NULL,
    op text DEFAULT ''::text NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: user_thirds_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.user_thirds_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_thirds_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.user_thirds_id_seq OWNED BY yinhe_app.user_thirds.id;


--
-- Name: user_tokens; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.user_tokens (
    id bigint NOT NULL,
    user_id bigint DEFAULT 0 NOT NULL,
    device_uuid text DEFAULT ''::text,
    device_id text DEFAULT ''::text,
    passkey_credential_id bigint DEFAULT 0 NOT NULL,
    token text DEFAULT ''::text NOT NULL,
    expired_at bigint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: user_tokens_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.user_tokens_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_tokens_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.user_tokens_id_seq OWNED BY yinhe_app.user_tokens.id;


--
-- Name: users; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.users (
    id bigint NOT NULL,
    username text DEFAULT ''::text NOT NULL,
    email text DEFAULT ''::text NOT NULL,
    phone text DEFAULT ''::text NOT NULL,
    password text DEFAULT ''::text NOT NULL,
    nickname text DEFAULT ''::text NOT NULL,
    avatar text DEFAULT ''::text NOT NULL,
    group_id bigint DEFAULT 0 NOT NULL,
    is_admin boolean DEFAULT false NOT NULL,
    role text DEFAULT 'user'::text NOT NULL,
    mfa_enabled boolean DEFAULT false NOT NULL,
    mfa_secret_encrypted text DEFAULT ''::text NOT NULL,
    mfa_backup_codes_encrypted text DEFAULT ''::text NOT NULL,
    web_authn_user_handle character varying(128) DEFAULT ''::character varying NOT NULL,
    status bigint DEFAULT 1 NOT NULL,
    remark text DEFAULT ''::text NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.users_id_seq OWNED BY yinhe_app.users.id;


--
-- Name: versions; Type: TABLE; Schema: yinhe_app; Owner: -
--

CREATE TABLE yinhe_app.versions (
    id bigint NOT NULL,
    version bigint DEFAULT 0 NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone
);


--
-- Name: versions_id_seq; Type: SEQUENCE; Schema: yinhe_app; Owner: -
--

CREATE SEQUENCE yinhe_app.versions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: versions_id_seq; Type: SEQUENCE OWNED BY; Schema: yinhe_app; Owner: -
--

ALTER SEQUENCE yinhe_app.versions_id_seq OWNED BY yinhe_app.versions.id;


--
-- Name: address_book_collection_rules id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.address_book_collection_rules ALTER COLUMN id SET DEFAULT nextval('yinhe_app.address_book_collection_rules_id_seq'::regclass);


--
-- Name: address_book_collections id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.address_book_collections ALTER COLUMN id SET DEFAULT nextval('yinhe_app.address_book_collections_id_seq'::regclass);


--
-- Name: address_books row_id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.address_books ALTER COLUMN row_id SET DEFAULT nextval('yinhe_app.address_books_row_id_seq'::regclass);


--
-- Name: audit_conns id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.audit_conns ALTER COLUMN id SET DEFAULT nextval('yinhe_app.audit_conns_id_seq'::regclass);


--
-- Name: audit_files id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.audit_files ALTER COLUMN id SET DEFAULT nextval('yinhe_app.audit_files_id_seq'::regclass);


--
-- Name: device_groups id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.device_groups ALTER COLUMN id SET DEFAULT nextval('yinhe_app.device_groups_id_seq'::regclass);


--
-- Name: groups id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.groups ALTER COLUMN id SET DEFAULT nextval('yinhe_app.groups_id_seq'::regclass);


--
-- Name: login_logs id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.login_logs ALTER COLUMN id SET DEFAULT nextval('yinhe_app.login_logs_id_seq'::regclass);


--
-- Name: oauths id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.oauths ALTER COLUMN id SET DEFAULT nextval('yinhe_app.oauths_id_seq'::regclass);


--
-- Name: passkey_credentials id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.passkey_credentials ALTER COLUMN id SET DEFAULT nextval('yinhe_app.passkey_credentials_id_seq'::regclass);


--
-- Name: peers row_id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.peers ALTER COLUMN row_id SET DEFAULT nextval('yinhe_app.peers_row_id_seq'::regclass);


--
-- Name: relay_nodes row_id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.relay_nodes ALTER COLUMN row_id SET DEFAULT nextval('yinhe_app.relay_nodes_row_id_seq'::regclass);


--
-- Name: server_cmds id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.server_cmds ALTER COLUMN id SET DEFAULT nextval('yinhe_app.server_cmds_id_seq'::regclass);


--
-- Name: share_records id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.share_records ALTER COLUMN id SET DEFAULT nextval('yinhe_app.share_records_id_seq'::regclass);


--
-- Name: tags id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.tags ALTER COLUMN id SET DEFAULT nextval('yinhe_app.tags_id_seq'::regclass);


--
-- Name: user_thirds id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.user_thirds ALTER COLUMN id SET DEFAULT nextval('yinhe_app.user_thirds_id_seq'::regclass);


--
-- Name: user_tokens id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.user_tokens ALTER COLUMN id SET DEFAULT nextval('yinhe_app.user_tokens_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.users ALTER COLUMN id SET DEFAULT nextval('yinhe_app.users_id_seq'::regclass);


--
-- Name: versions id; Type: DEFAULT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.versions ALTER COLUMN id SET DEFAULT nextval('yinhe_app.versions_id_seq'::regclass);


--
-- Name: address_book_collection_rules address_book_collection_rules_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.address_book_collection_rules
    ADD CONSTRAINT address_book_collection_rules_pkey PRIMARY KEY (id);


--
-- Name: address_book_collections address_book_collections_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.address_book_collections
    ADD CONSTRAINT address_book_collections_pkey PRIMARY KEY (id);


--
-- Name: address_books address_books_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.address_books
    ADD CONSTRAINT address_books_pkey PRIMARY KEY (row_id);


--
-- Name: audit_conns audit_conns_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.audit_conns
    ADD CONSTRAINT audit_conns_pkey PRIMARY KEY (id);


--
-- Name: audit_files audit_files_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.audit_files
    ADD CONSTRAINT audit_files_pkey PRIMARY KEY (id);


--
-- Name: device_groups device_groups_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.device_groups
    ADD CONSTRAINT device_groups_pkey PRIMARY KEY (id);


--
-- Name: groups groups_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.groups
    ADD CONSTRAINT groups_pkey PRIMARY KEY (id);


--
-- Name: login_logs login_logs_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.login_logs
    ADD CONSTRAINT login_logs_pkey PRIMARY KEY (id);


--
-- Name: oauths oauths_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.oauths
    ADD CONSTRAINT oauths_pkey PRIMARY KEY (id);


--
-- Name: passkey_credentials passkey_credentials_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.passkey_credentials
    ADD CONSTRAINT passkey_credentials_pkey PRIMARY KEY (id);


--
-- Name: peers peers_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.peers
    ADD CONSTRAINT peers_pkey PRIMARY KEY (row_id);


--
-- Name: relay_nodes relay_nodes_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.relay_nodes
    ADD CONSTRAINT relay_nodes_pkey PRIMARY KEY (row_id);


--
-- Name: server_cmds server_cmds_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.server_cmds
    ADD CONSTRAINT server_cmds_pkey PRIMARY KEY (id);


--
-- Name: share_records share_records_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.share_records
    ADD CONSTRAINT share_records_pkey PRIMARY KEY (id);


--
-- Name: tags tags_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (id);


--
-- Name: user_thirds user_thirds_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.user_thirds
    ADD CONSTRAINT user_thirds_pkey PRIMARY KEY (id);


--
-- Name: user_tokens user_tokens_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.user_tokens
    ADD CONSTRAINT user_tokens_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: versions versions_pkey; Type: CONSTRAINT; Schema: yinhe_app; Owner: -
--

ALTER TABLE ONLY yinhe_app.versions
    ADD CONSTRAINT versions_pkey PRIMARY KEY (id);


--
-- Name: idx_address_book_collection_rules_collection_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_address_book_collection_rules_collection_id ON yinhe_app.address_book_collection_rules USING btree (collection_id);


--
-- Name: idx_address_book_collections_user_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_address_book_collections_user_id ON yinhe_app.address_book_collections USING btree (user_id);


--
-- Name: idx_address_books_collection_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_address_books_collection_id ON yinhe_app.address_books USING btree (collection_id);


--
-- Name: idx_address_books_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_address_books_id ON yinhe_app.address_books USING btree (id);


--
-- Name: idx_address_books_user_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_address_books_user_id ON yinhe_app.address_books USING btree (user_id);


--
-- Name: idx_audit_conns_conn_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_audit_conns_conn_id ON yinhe_app.audit_conns USING btree (conn_id);


--
-- Name: idx_audit_conns_peer_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_audit_conns_peer_id ON yinhe_app.audit_conns USING btree (peer_id);


--
-- Name: idx_audit_files_from_peer; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_audit_files_from_peer ON yinhe_app.audit_files USING btree (from_peer);


--
-- Name: idx_audit_files_peer_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_audit_files_peer_id ON yinhe_app.audit_files USING btree (peer_id);


--
-- Name: idx_login_logs_passkey_credential_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_login_logs_passkey_credential_id ON yinhe_app.login_logs USING btree (passkey_credential_id);


--
-- Name: idx_passkey_credential_rp; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE UNIQUE INDEX idx_passkey_credential_rp ON yinhe_app.passkey_credentials USING btree (rp_id, credential_id);


--
-- Name: idx_passkey_credentials_user_handle; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_passkey_credentials_user_handle ON yinhe_app.passkey_credentials USING btree (user_handle);


--
-- Name: idx_passkey_user_rp; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_passkey_user_rp ON yinhe_app.passkey_credentials USING btree (user_id, rp_id);


--
-- Name: idx_peers_alias; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_peers_alias ON yinhe_app.peers USING btree (alias);


--
-- Name: idx_peers_group_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_peers_group_id ON yinhe_app.peers USING btree (group_id);


--
-- Name: idx_peers_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_peers_id ON yinhe_app.peers USING btree (id);


--
-- Name: idx_peers_status; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_peers_status ON yinhe_app.peers USING btree (status);


--
-- Name: idx_peers_user_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_peers_user_id ON yinhe_app.peers USING btree (user_id);


--
-- Name: idx_peers_uuid; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_peers_uuid ON yinhe_app.peers USING btree (uuid);


--
-- Name: idx_relay_nodes_name; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_relay_nodes_name ON yinhe_app.relay_nodes USING btree (name);


--
-- Name: idx_relay_nodes_status; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_relay_nodes_status ON yinhe_app.relay_nodes USING btree (status);


--
-- Name: idx_share_records_peer_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_share_records_peer_id ON yinhe_app.share_records USING btree (peer_id);


--
-- Name: idx_share_records_share_token; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_share_records_share_token ON yinhe_app.share_records USING btree (share_token);


--
-- Name: idx_share_records_user_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_share_records_user_id ON yinhe_app.share_records USING btree (user_id);


--
-- Name: idx_tags_collection_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_tags_collection_id ON yinhe_app.tags USING btree (collection_id);


--
-- Name: idx_tags_user_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_tags_user_id ON yinhe_app.tags USING btree (user_id);


--
-- Name: idx_user_thirds_open_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_user_thirds_open_id ON yinhe_app.user_thirds USING btree (open_id);


--
-- Name: idx_user_thirds_user_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_user_thirds_user_id ON yinhe_app.user_thirds USING btree (user_id);


--
-- Name: idx_user_tokens_passkey_credential_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_user_tokens_passkey_credential_id ON yinhe_app.user_tokens USING btree (passkey_credential_id);


--
-- Name: idx_user_tokens_token; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_user_tokens_token ON yinhe_app.user_tokens USING btree (token);


--
-- Name: idx_user_tokens_user_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_user_tokens_user_id ON yinhe_app.user_tokens USING btree (user_id);


--
-- Name: idx_users_email; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_users_email ON yinhe_app.users USING btree (email);


--
-- Name: idx_users_group_id; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_users_group_id ON yinhe_app.users USING btree (group_id);


--
-- Name: idx_users_mfa_enabled; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_users_mfa_enabled ON yinhe_app.users USING btree (mfa_enabled);


--
-- Name: idx_users_phone; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_users_phone ON yinhe_app.users USING btree (phone);


--
-- Name: idx_users_role; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_users_role ON yinhe_app.users USING btree (role);


--
-- Name: idx_users_username; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE UNIQUE INDEX idx_users_username ON yinhe_app.users USING btree (username);


--
-- Name: idx_users_web_authn_user_handle; Type: INDEX; Schema: yinhe_app; Owner: -
--

CREATE INDEX idx_users_web_authn_user_handle ON yinhe_app.users USING btree (web_authn_user_handle);


-- Record the exact application schema version. Bootstrap users and imported
-- business data are intentionally not part of this schema migration.
INSERT INTO yinhe_app.versions (version, created_at, updated_at)
SELECT 272, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM yinhe_app.versions WHERE version = 272);

-- Private-schema defense in depth. The application role is trusted to enforce
-- user/tenant authorization, while PostgreSQL still constrains its object-level
-- capabilities and prevents direct Data API access.
do $rls$
declare
    table_name text;
begin
    foreach table_name in array array[
        'address_book_collection_rules', 'address_book_collections', 'address_books',
        'device_groups', 'groups', 'login_logs', 'oauths', 'passkey_credentials',
        'peers', 'relay_nodes', 'server_cmds', 'share_records', 'tags',
        'user_thirds', 'user_tokens', 'users'
    ] loop
        execute format('alter table yinhe_app.%I enable row level security', table_name);
        execute format(
            'create policy yinhe_app_runtime_all on yinhe_app.%I for all to yinhe_app_runtime using (true) with check (true)',
            table_name
        );
    end loop;

    foreach table_name in array array['audit_conns', 'audit_files'] loop
        execute format('alter table yinhe_app.%I enable row level security', table_name);
        execute format(
            'create policy yinhe_app_runtime_select on yinhe_app.%I for select to yinhe_app_runtime using (true)',
            table_name
        );
        execute format(
            'create policy yinhe_app_runtime_insert on yinhe_app.%I for insert to yinhe_app_runtime with check (true)',
            table_name
        );
    end loop;

    alter table yinhe_app.versions enable row level security;
    create policy yinhe_app_runtime_select on yinhe_app.versions
        for select to yinhe_app_runtime using (true);
end
$rls$;

grant select, insert, update, delete on all tables in schema yinhe_app to yinhe_app_runtime;
revoke update, delete on yinhe_app.audit_conns, yinhe_app.audit_files from yinhe_app_runtime;
revoke insert, update, delete on yinhe_app.versions from yinhe_app_runtime;
grant usage, select on all sequences in schema yinhe_app to yinhe_app_runtime;

revoke all on all tables in schema yinhe_app from public, anon, authenticated, service_role;
revoke all on all sequences in schema yinhe_app from public, anon, authenticated, service_role;
revoke all on all functions in schema yinhe_app from public, anon, authenticated, service_role;

alter default privileges for role postgres in schema yinhe_app
    revoke all on tables from public, anon, authenticated, service_role;
alter default privileges for role postgres in schema yinhe_app
    revoke all on sequences from public, anon, authenticated, service_role;
alter default privileges for role postgres in schema yinhe_app
    revoke all on functions from public, anon, authenticated, service_role;
alter default privileges for role postgres in schema yinhe_app
    grant select, insert, update, delete on tables to yinhe_app_runtime;
alter default privileges for role postgres in schema yinhe_app
    grant usage, select on sequences to yinhe_app_runtime;

-- Some older Supabase projects retain supabase_admin default ACLs. Tighten
-- those defaults only when the migration role is permitted to manage them.
do $default_acl$
begin
    if pg_has_role(current_user, 'supabase_admin', 'MEMBER') then
        execute 'alter default privileges for role supabase_admin in schema yinhe_app revoke all on tables from public, anon, authenticated, service_role';
        execute 'alter default privileges for role supabase_admin in schema yinhe_app revoke all on sequences from public, anon, authenticated, service_role';
        execute 'alter default privileges for role supabase_admin in schema yinhe_app revoke all on functions from public, anon, authenticated, service_role';
    end if;
end
$default_acl$;

commit;


--
-- PostgreSQL database dump complete
--
