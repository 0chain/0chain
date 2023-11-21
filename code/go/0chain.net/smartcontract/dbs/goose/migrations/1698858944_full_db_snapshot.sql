-- +goose Up
-- +goose StatementBegin
 SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: ltree; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS ltree WITH SCHEMA public;


--
-- Name: EXTENSION ltree; Type: COMMENT; Schema: -; Owner: 
--

COMMENT ON EXTENSION ltree IS 'data type for hierarchical tree-like structures';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: allocation_blobber_terms; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE allocation_blobber_terms (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    alloc_id bigint NOT NULL,
    blobber_id text NOT NULL,
    read_price bigint,
    write_price bigint,
    alloc_blobber_idx numeric,
    max_offer_duration bigint
);


ALTER TABLE allocation_blobber_terms OWNER TO zchain_user;

--
-- Name: allocation_blobber_terms_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE allocation_blobber_terms_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE allocation_blobber_terms_id_seq OWNER TO zchain_user;

--
-- Name: allocation_blobber_terms_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE allocation_blobber_terms_id_seq OWNED BY allocation_blobber_terms.id;


--
-- Name: allocations; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE allocations (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    allocation_id text,
    allocation_name character varying(64),
    transaction_id text,
    data_shards bigint,
    parity_shards bigint,
    size bigint,
    expiration bigint,
    owner text,
    owner_public_key text,
    read_price_min bigint,
    read_price_max bigint,
    write_price_min bigint,
    write_price_max bigint,
    start_time bigint,
    finalized boolean,
    cancelled boolean,
    used_size bigint,
    moved_to_challenge bigint,
    moved_back bigint,
    moved_to_validators bigint,
    time_unit bigint,
    num_writes bigint,
    num_reads bigint,
    total_challenges bigint,
    open_challenges bigint,
    successful_challenges bigint,
    failed_challenges bigint,
    latest_closed_challenge_txn text,
    write_pool bigint,
    third_party_extendable boolean DEFAULT false,
    file_options smallint DEFAULT 63,
);


ALTER TABLE allocations OWNER TO zchain_user;

--
-- Name: allocations_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE allocations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE allocations_id_seq OWNER TO zchain_user;

--
-- Name: allocations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE allocations_id_seq OWNED BY allocations.id;


--
-- Name: authorizer_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE authorizer_aggregates (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    authorizer_id text,
    round bigint NOT NULL,
    fee bigint,
    total_stake bigint,
    total_rewards bigint,
    service_charge numeric,
    total_mint bigint DEFAULT 0 NOT NULL,
    total_burn bigint DEFAULT 0 NOT NULL
)
PARTITION BY RANGE (round);


ALTER TABLE authorizer_aggregates OWNER TO zchain_user;

--
-- Name: authorizer_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE authorizer_aggregates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE authorizer_aggregates_id_seq OWNER TO zchain_user;

--
-- Name: authorizer_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE authorizer_aggregates_id_seq OWNED BY authorizer_aggregates.id;

--
-- Name: authorizer_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE authorizer_snapshots (
    authorizer_id text,
    round bigint,
    fee bigint,
    total_stake bigint,
    total_rewards bigint,
    service_charge numeric,
    creation_round bigint,
    total_mint bigint DEFAULT 0 NOT NULL,
    total_burn bigint DEFAULT 0 NOT NULL,
    is_killed boolean DEFAULT false NOT NULL,
    is_shutdown boolean DEFAULT false NOT NULL
);


ALTER TABLE authorizer_snapshots OWNER TO zchain_user;

--
-- Name: authorizers; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE authorizers (
    id text NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    delegate_wallet text,
    min_stake bigint,
    max_stake bigint,
    num_delegates bigint,
    service_charge numeric,
    total_stake bigint,
    downtime bigint,
    last_health_check bigint,
    url text,
    fee bigint,
    creation_round bigint,
    is_killed boolean,
    is_shutdown boolean,
    total_mint bigint DEFAULT 0 NOT NULL,
    total_burn bigint DEFAULT 0 NOT NULL
);


ALTER TABLE authorizers OWNER TO zchain_user;

--
-- Name: blobber_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE blobber_aggregates (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    blobber_id text,
    round bigint NOT NULL,
    write_price bigint,
    capacity bigint,
    allocated bigint,
    saved_data bigint,
    read_data bigint,
    offers_total bigint,
    total_stake bigint,
    total_service_charge bigint,
    total_rewards bigint,
    challenges_passed bigint,
    challenges_completed bigint,
    open_challenges bigint,
    inactive_rounds bigint,
    rank_metric numeric,
    downtime bigint,
    total_storage_income bigint DEFAULT 0 NOT NULL,
    total_read_income bigint DEFAULT 0 NOT NULL,
    total_slashed_stake bigint DEFAULT 0 NOT NULL,
    total_block_rewards bigint DEFAULT 0 NOT NULL
)
PARTITION BY RANGE (round);


ALTER TABLE blobber_aggregates OWNER TO zchain_user;

--
-- Name: blobber_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE blobber_aggregates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE blobber_aggregates_id_seq OWNER TO zchain_user;

--
-- Name: blobber_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE blobber_aggregates_id_seq OWNED BY blobber_aggregates.id;


--
-- Name: blobber_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE blobber_snapshots (
    blobber_id text,
    write_price bigint,
    capacity bigint,
    allocated bigint,
    saved_data bigint,
    read_data bigint,
    offers_total bigint,
    total_service_charge bigint,
    total_rewards bigint,
    total_stake bigint,
    challenges_passed bigint,
    challenges_completed bigint,
    open_challenges bigint,
    creation_round bigint,
    rank_metric numeric,
    bucket_id bigint DEFAULT 0,
    total_storage_income bigint DEFAULT 0 NOT NULL,
    total_read_income bigint DEFAULT 0 NOT NULL,
    total_slashed_stake bigint DEFAULT 0 NOT NULL,
    total_block_rewards bigint DEFAULT 0 NOT NULL,
    is_killed boolean DEFAULT false NOT NULL,
    is_shutdown boolean DEFAULT false NOT NULL,
    round bigint DEFAULT 0 NOT NULL
);


ALTER TABLE blobber_snapshots OWNER TO zchain_user;

--
-- Name: blobbers; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE blobbers (
    id text NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    delegate_wallet text,
    min_stake bigint,
    max_stake bigint,
    num_delegates bigint,
    service_charge numeric,
    total_stake bigint,
    downtime bigint,
    last_health_check bigint,
    base_url text,
    read_price bigint,
    write_price bigint,
    max_offer_duration bigint,
    capacity bigint,
    allocated bigint,
    used bigint,
    saved_data bigint,
    read_data bigint,
    offers_total bigint,
    total_service_charge bigint,
    name text,
    website_url text,
    logo_url text,
    description text,
    challenges_passed bigint,
    challenges_completed bigint,
    open_challenges bigint,
    rank_metric numeric,
    creation_round bigint,
    is_killed boolean,
    is_shutdown boolean,
    total_storage_income bigint DEFAULT 0 NOT NULL,
    total_read_income bigint DEFAULT 0 NOT NULL,
    total_slashed_stake bigint DEFAULT 0 NOT NULL,
    is_available boolean,
    total_block_rewards bigint DEFAULT 0 NOT NULL,
    not_available boolean
);


ALTER TABLE blobbers OWNER TO zchain_user;

--
-- Name: blocks; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE blocks (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    hash text,
    version text,
    creation_date bigint,
    round bigint NOT NULL,
    miner_id text,
    round_random_seed bigint,
    merkle_tree_root text,
    state_hash text,
    receipt_merkle_tree_root text,
    num_txns bigint,
    magic_block_hash text,
    prev_hash text,
    signature text,
    chain_id text,
    state_changes_count bigint,
    running_txn_count text,
    round_timeout_count bigint
)
PARTITION BY RANGE (round);


ALTER TABLE blocks OWNER TO zchain_user;

--
-- Name: blocks_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE blocks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE blocks_id_seq OWNER TO zchain_user;

--
-- Name: blocks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE blocks_id_seq OWNED BY blocks.id;

--
-- Name: burn_tickets; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE burn_tickets (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    ethereum_address text,
    hash text,
    nonce bigint,
    amount bigint
);


ALTER TABLE burn_tickets OWNER TO zchain_user;

--
-- Name: burn_tickets_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE burn_tickets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE burn_tickets_id_seq OWNER TO zchain_user;

--
-- Name: burn_tickets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE burn_tickets_id_seq OWNED BY burn_tickets.id;


--
-- Name: challenge_pools; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE challenge_pools (
    id text NOT NULL,
    allocation_id text,
    balance bigint,
    start_time bigint,
    expiration bigint,
    finalized boolean
);


ALTER TABLE challenge_pools OWNER TO zchain_user;

--
-- Name: challenges; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE challenges (
    id bigint NOT NULL,
    created_at bigint,
    updated_at timestamp with time zone,
    challenge_id text,
    allocation_id text,
    blobber_id text,
    validators_id text,
    seed bigint,
    allocation_root text,
    responded bigint,
    passed boolean,
    round_created_at bigint,
    round_responded bigint,
    "timestamp" bigint
);


ALTER TABLE challenges OWNER TO zchain_user;

--
-- Name: challenges_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE challenges_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE challenges_id_seq OWNER TO zchain_user;

--
-- Name: challenges_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE challenges_id_seq OWNED BY challenges.id;


--
-- Name: delegate_pools; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE delegate_pools (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    pool_id text,
    provider_type bigint,
    provider_id text,
    delegate_id text,
    balance bigint,
    reward bigint,
    total_reward bigint,
    total_penalty bigint,
    status bigint,
    round_created bigint,
    round_pool_last_updated bigint,
    staked_at bigint
);


ALTER TABLE delegate_pools OWNER TO zchain_user;

--
-- Name: delegate_pools_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE delegate_pools_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE delegate_pools_id_seq OWNER TO zchain_user;

--
-- Name: delegate_pools_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE delegate_pools_id_seq OWNED BY delegate_pools.id;


--
-- Name: errors; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE errors (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    transaction_id text,
    error text
);


ALTER TABLE errors OWNER TO zchain_user;

--
-- Name: errors_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE errors_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE errors_id_seq OWNER TO zchain_user;

--
-- Name: errors_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE errors_id_seq OWNED BY errors.id;


--
-- Name: events; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE events (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    block_number bigint NOT NULL,
    tx_hash text,
    type bigint,
    tag bigint,
    index text
)
PARTITION BY RANGE (block_number);


ALTER TABLE events OWNER TO zchain_user;

--
-- Name: events_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE events_id_seq OWNER TO zchain_user;

--
-- Name: events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE events_id_seq OWNED BY events.id;

--
-- Name: miner_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE miner_aggregates (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    miner_id text,
    round bigint NOT NULL,
    fees bigint,
    total_stake bigint,
    total_rewards bigint,
    service_charge numeric
)
PARTITION BY RANGE (round);


ALTER TABLE miner_aggregates OWNER TO zchain_user;

--
-- Name: miner_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE miner_aggregates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE miner_aggregates_id_seq OWNER TO zchain_user;

--
-- Name: miner_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE miner_aggregates_id_seq OWNED BY miner_aggregates.id;

--
-- Name: miner_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE miner_snapshots (
    miner_id text,
    round bigint,
    fees bigint,
    total_stake bigint,
    total_rewards bigint,
    service_charge numeric,
    creation_round bigint,
    is_killed boolean DEFAULT false NOT NULL,
    is_shutdown boolean DEFAULT false NOT NULL
);


ALTER TABLE miner_snapshots OWNER TO zchain_user;

--
-- Name: miners; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE miners (
    id text NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    delegate_wallet text,
    min_stake bigint,
    max_stake bigint,
    num_delegates bigint,
    service_charge numeric,
    total_stake bigint,
    downtime bigint,
    last_health_check bigint,
    n2n_host text,
    host text,
    port bigint,
    path text,
    public_key text,
    short_name text,
    build_tag text,
    delete boolean,
    fees bigint,
    active boolean,
    creation_round bigint,
    is_killed boolean,
    is_shutdown boolean
);


ALTER TABLE miners OWNER TO zchain_user;

--
-- Name: provider_rewards; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE provider_rewards (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    provider_id text,
    rewards bigint,
    total_rewards bigint,
    round_service_charge_last_updated bigint
);


ALTER TABLE provider_rewards OWNER TO zchain_user;

--
-- Name: provider_rewards_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE provider_rewards_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE provider_rewards_id_seq OWNER TO zchain_user;

--
-- Name: provider_rewards_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE provider_rewards_id_seq OWNED BY provider_rewards.id;


--
-- Name: read_markers; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE read_markers (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    client_id text,
    blobber_id text,
    allocation_id text,
    transaction_id text,
    owner_id text,
    "timestamp" bigint,
    read_counter bigint,
    read_size numeric,
    signature text,
    payer_id text,
    auth_ticket text,
    block_number bigint
);


ALTER TABLE read_markers OWNER TO zchain_user;

--
-- Name: read_markers_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE read_markers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE read_markers_id_seq OWNER TO zchain_user;

--
-- Name: read_markers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE read_markers_id_seq OWNED BY read_markers.id;


--
-- Name: read_pools; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE read_pools (
    id bigint,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    user_id text NOT NULL,
    balance bigint
);


ALTER TABLE read_pools OWNER TO zchain_user;

--
-- Name: read_pools_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE read_pools_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE read_pools_id_seq OWNER TO zchain_user;

--
-- Name: read_pools_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE read_pools_id_seq OWNED BY read_pools.id;


--
-- Name: reward_delegates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE reward_delegates (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    amount bigint,
    block_number bigint,
    pool_id text,
    reward_type bigint,
    allocation_id text,
    provider_id text
);


ALTER TABLE reward_delegates OWNER TO zchain_user;

--
-- Name: reward_delegates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE reward_delegates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE reward_delegates_id_seq OWNER TO zchain_user;

--
-- Name: reward_delegates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE reward_delegates_id_seq OWNED BY reward_delegates.id;


--
-- Name: reward_mints; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE reward_mints (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    amount bigint,
    block_number bigint,
    client_id text,
    pool_id text,
    provider_type text,
    provider_id text
);


ALTER TABLE reward_mints OWNER TO zchain_user;

--
-- Name: reward_mints_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE reward_mints_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE reward_mints_id_seq OWNER TO zchain_user;

--
-- Name: reward_mints_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE reward_mints_id_seq OWNED BY reward_mints.id;


--
-- Name: reward_providers; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE reward_providers (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    amount bigint,
    block_number bigint,
    provider_id text,
    reward_type bigint,
    allocation_id text
);


ALTER TABLE reward_providers OWNER TO zchain_user;

--
-- Name: reward_providers_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE reward_providers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE reward_providers_id_seq OWNER TO zchain_user;

--
-- Name: reward_providers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE reward_providers_id_seq OWNED BY reward_providers.id;


--
-- Name: sharder_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE sharder_aggregates (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    sharder_id text,
    round bigint NOT NULL,
    fees bigint,
    total_stake bigint,
    total_rewards bigint,
    service_charge numeric
)
PARTITION BY RANGE (round);


ALTER TABLE sharder_aggregates OWNER TO zchain_user;

--
-- Name: sharder_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE sharder_aggregates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE sharder_aggregates_id_seq OWNER TO zchain_user;

--
-- Name: sharder_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE sharder_aggregates_id_seq OWNED BY sharder_aggregates.id;

--
-- Name: sharder_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE sharder_snapshots (
    sharder_id text,
    round bigint,
    fees bigint,
    total_stake bigint,
    total_rewards bigint,
    service_charge numeric,
    creation_round bigint,
    is_killed boolean DEFAULT false NOT NULL,
    is_shutdown boolean DEFAULT false NOT NULL
);


ALTER TABLE sharder_snapshots OWNER TO zchain_user;

--
-- Name: sharders; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE sharders (
    id text NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    delegate_wallet text,
    min_stake bigint,
    max_stake bigint,
    num_delegates bigint,
    service_charge numeric,
    total_stake bigint,
    downtime bigint,
    last_health_check bigint,
    n2n_host text,
    host text,
    port bigint,
    path text,
    public_key text,
    short_name text,
    build_tag text,
    delete boolean,
    fees bigint,
    active boolean,
    creation_round bigint,
    is_killed boolean,
    is_shutdown boolean
);


ALTER TABLE sharders OWNER TO zchain_user;

--
-- Name: snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE snapshots (
    round bigint NOT NULL,
    total_mint bigint,
    total_challenge_pools bigint,
    active_allocated_delta bigint,
    zcn_supply bigint,
    total_value_locked bigint,
    client_locks bigint,
    mined_total bigint,
    total_staked bigint,
    total_rewards bigint,
    successful_challenges bigint,
    total_challenges bigint,
    allocated_storage bigint,
    max_capacity_storage bigint,
    staked_storage bigint,
    used_storage bigint,
    transactions_count bigint,
    unique_addresses bigint,
    block_count bigint,
    created_at bigint,
    miner_count bigint DEFAULT 0,
    sharder_count bigint DEFAULT 0,
    blobber_count bigint DEFAULT 0,
    authorizer_count bigint DEFAULT 0,
    validator_count bigint DEFAULT 0,
    total_txn_fee bigint DEFAULT 0,
    blobbers_stake bigint DEFAULT 0,
    storage_token_stake bigint DEFAULT 0 NOT NULL,
    miner_total_rewards bigint DEFAULT 0 NOT NULL,
    sharder_total_rewards bigint DEFAULT 0 NOT NULL,
    blobber_total_rewards bigint DEFAULT 0 NOT NULL,
    total_read_pool_locked bigint DEFAULT 0 NOT NULL
)
PARTITION BY RANGE (round);


ALTER TABLE snapshots OWNER TO zchain_user;

--
-- Name: transaction_errors; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE transaction_errors (
    id integer NOT NULL,
    created_at timestamp with time zone,
    transaction_output text,
    count bigint
);


ALTER TABLE transaction_errors OWNER TO zchain_user;

--
-- Name: transaction_errors_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE transaction_errors_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE transaction_errors_id_seq OWNER TO zchain_user;

--
-- Name: transaction_errors_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE transaction_errors_id_seq OWNED BY transaction_errors.id;


--
-- Name: transactions; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE transactions (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    hash text,
    block_hash text,
    round bigint NOT NULL,
    version text,
    client_id text,
    to_client_id text,
    transaction_data text,
    value bigint,
    signature text,
    creation_date bigint,
    fee bigint,
    nonce bigint,
    transaction_type bigint,
    transaction_output text,
    output_hash text,
    status bigint
)
PARTITION BY RANGE (round);


ALTER TABLE transactions OWNER TO zchain_user;

--
-- Name: transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE transactions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE transactions_id_seq OWNER TO zchain_user;

--
-- Name: transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE transactions_id_seq OWNED BY transactions.id;

--
-- Name: user_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE user_aggregates (
    user_id text NOT NULL,
    round bigint NOT NULL,
    collected_reward bigint,
    total_stake bigint,
    read_pool_total bigint,
    write_pool_total bigint,
    payed_fees bigint,
    created_at timestamp with time zone
)
PARTITION BY RANGE (round);


ALTER TABLE user_aggregates OWNER TO zchain_user;

--
-- Name: user_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE user_snapshots (
    user_id text,
    round bigint,
    collected_reward bigint,
    total_stake bigint,
    read_pool_total bigint,
    write_pool_total bigint,
    payed_fees bigint,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);


ALTER TABLE user_snapshots OWNER TO zchain_user;

--
-- Name: users; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE users (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    user_id text,
    txn_hash text,
    balance bigint,
    change bigint,
    round bigint,
    nonce bigint,
    mint_nonce bigint
);


ALTER TABLE users OWNER TO zchain_user;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE users_id_seq OWNER TO zchain_user;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE users_id_seq OWNED BY users.id;


--
-- Name: validator_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE validator_aggregates (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    validator_id text,
    round bigint NOT NULL,
    total_stake bigint,
    total_rewards bigint,
    service_charge numeric
)
PARTITION BY RANGE (round);


ALTER TABLE validator_aggregates OWNER TO zchain_user;

--
-- Name: validator_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE validator_aggregates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE validator_aggregates_id_seq OWNER TO zchain_user;

--
-- Name: validator_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE validator_aggregates_id_seq OWNED BY validator_aggregates.id;

--
-- Name: validator_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE validator_snapshots (
    validator_id text,
    total_stake bigint,
    total_rewards bigint,
    service_charge numeric,
    creation_round bigint,
    is_killed boolean DEFAULT false NOT NULL,
    is_shutdown boolean DEFAULT false NOT NULL,
    round bigint DEFAULT 0 NOT NULL
);


ALTER TABLE validator_snapshots OWNER TO zchain_user;

--
-- Name: validators; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE validators (
    id text NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    delegate_wallet text,
    min_stake bigint,
    max_stake bigint,
    num_delegates bigint,
    service_charge numeric,
    total_stake bigint,
    downtime bigint,
    last_health_check bigint,
    base_url text,
    public_key text,
    creation_round bigint,
    is_killed boolean,
    is_shutdown boolean
);


ALTER TABLE validators OWNER TO zchain_user;

--
-- Name: write_markers; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE write_markers (
    id bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    client_id text,
    blobber_id text,
    allocation_id text,
    transaction_id text,
    allocation_root text,
    file_meta_root character(64),
    previous_allocation_root text,
    size bigint,
    "timestamp" bigint,
    signature text,
    block_number bigint
);


ALTER TABLE write_markers OWNER TO zchain_user;

--
-- Name: write_markers_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE write_markers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE write_markers_id_seq OWNER TO zchain_user;

--
-- Name: write_markers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE write_markers_id_seq OWNED BY write_markers.id;

--
-- Name: allocation_blobber_terms id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY allocation_blobber_terms ALTER COLUMN id SET DEFAULT nextval('allocation_blobber_terms_id_seq'::regclass);


--
-- Name: allocations id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY allocations ALTER COLUMN id SET DEFAULT nextval('allocations_id_seq'::regclass);


--
-- Name: authorizer_aggregates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY authorizer_aggregates ALTER COLUMN id SET DEFAULT nextval('authorizer_aggregates_id_seq'::regclass);


--
-- Name: blobber_aggregates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY blobber_aggregates ALTER COLUMN id SET DEFAULT nextval('blobber_aggregates_id_seq'::regclass);


--
-- Name: blocks id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY blocks ALTER COLUMN id SET DEFAULT nextval('blocks_id_seq'::regclass);


--
-- Name: burn_tickets id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY burn_tickets ALTER COLUMN id SET DEFAULT nextval('burn_tickets_id_seq'::regclass);


--
-- Name: challenges id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY challenges ALTER COLUMN id SET DEFAULT nextval('challenges_id_seq'::regclass);


--
-- Name: delegate_pools id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY delegate_pools ALTER COLUMN id SET DEFAULT nextval('delegate_pools_id_seq'::regclass);


--
-- Name: errors id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY errors ALTER COLUMN id SET DEFAULT nextval('errors_id_seq'::regclass);


--
-- Name: events id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY events ALTER COLUMN id SET DEFAULT nextval('events_id_seq'::regclass);



--
-- Name: miner_aggregates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY miner_aggregates ALTER COLUMN id SET DEFAULT nextval('miner_aggregates_id_seq'::regclass);


--
-- Name: provider_rewards id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY provider_rewards ALTER COLUMN id SET DEFAULT nextval('provider_rewards_id_seq'::regclass);


--
-- Name: read_markers id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY read_markers ALTER COLUMN id SET DEFAULT nextval('read_markers_id_seq'::regclass);


--
-- Name: reward_delegates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY reward_delegates ALTER COLUMN id SET DEFAULT nextval('reward_delegates_id_seq'::regclass);


--
-- Name: reward_mints id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY reward_mints ALTER COLUMN id SET DEFAULT nextval('reward_mints_id_seq'::regclass);


--
-- Name: reward_providers id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY reward_providers ALTER COLUMN id SET DEFAULT nextval('reward_providers_id_seq'::regclass);


--
-- Name: sharder_aggregates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY sharder_aggregates ALTER COLUMN id SET DEFAULT nextval('sharder_aggregates_id_seq'::regclass);


--
-- Name: transaction_errors id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY transaction_errors ALTER COLUMN id SET DEFAULT nextval('transaction_errors_id_seq'::regclass);


--
-- Name: transactions id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY transactions ALTER COLUMN id SET DEFAULT nextval('transactions_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);


--
-- Name: validator_aggregates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY validator_aggregates ALTER COLUMN id SET DEFAULT nextval('validator_aggregates_id_seq'::regclass);


--
-- Name: write_markers id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY write_markers ALTER COLUMN id SET DEFAULT nextval('write_markers_id_seq'::regclass);


--
-- Name: allocation_blobber_terms allocation_blobber_terms_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY allocation_blobber_terms
    ADD CONSTRAINT allocation_blobber_terms_pkey PRIMARY KEY (id);


--
-- Name: allocations allocations_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY allocations
    ADD CONSTRAINT allocations_pkey PRIMARY KEY (id);


--
-- Name: authorizer_aggregates authorizer_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY authorizer_aggregates
    ADD CONSTRAINT authorizer_aggregates_pkey PRIMARY KEY (id, round);


--
-- Name: authorizers authorizers_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY authorizers
    ADD CONSTRAINT authorizers_pkey PRIMARY KEY (id);


--
-- Name: blobber_aggregates blobber_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY blobber_aggregates
    ADD CONSTRAINT blobber_aggregates_pkey PRIMARY KEY (id, round);

--
-- Name: blobbers blobbers_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY blobbers
    ADD CONSTRAINT blobbers_pkey PRIMARY KEY (id);


--
-- Name: blocks blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY blocks
    ADD CONSTRAINT blocks_pkey PRIMARY KEY (id, round);

--
-- Name: challenge_pools challenge_pools_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY challenge_pools
    ADD CONSTRAINT challenge_pools_pkey PRIMARY KEY (id);


--
-- Name: challenges challenges_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY challenges
    ADD CONSTRAINT challenges_pkey PRIMARY KEY (id);


--
-- Name: delegate_pools delegate_pools_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY delegate_pools
    ADD CONSTRAINT delegate_pools_pkey PRIMARY KEY (id);


--
-- Name: errors errors_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY errors
    ADD CONSTRAINT errors_pkey PRIMARY KEY (id);


--
-- Name: events events_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY events
    ADD CONSTRAINT events_pkey PRIMARY KEY (id, block_number);

--
-- Name: miner_aggregates miner_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY miner_aggregates
    ADD CONSTRAINT miner_aggregates_pkey PRIMARY KEY (id, round);

--
-- Name: miners miners_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY miners
    ADD CONSTRAINT miners_pkey PRIMARY KEY (id);


--
-- Name: provider_rewards provider_rewards_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY provider_rewards
    ADD CONSTRAINT provider_rewards_pkey PRIMARY KEY (id);


--
-- Name: read_markers read_markers_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY read_markers
    ADD CONSTRAINT read_markers_pkey PRIMARY KEY (id);


--
-- Name: read_pools read_pools_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY read_pools
    ADD CONSTRAINT read_pools_pkey PRIMARY KEY (user_id);


--
-- Name: reward_delegates reward_delegates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY reward_delegates
    ADD CONSTRAINT reward_delegates_pkey PRIMARY KEY (id);


--
-- Name: reward_mints reward_mints_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY reward_mints
    ADD CONSTRAINT reward_mints_pkey PRIMARY KEY (id);


--
-- Name: reward_providers reward_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY reward_providers
    ADD CONSTRAINT reward_providers_pkey PRIMARY KEY (id);


--
-- Name: sharder_aggregates sharder_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY sharder_aggregates
    ADD CONSTRAINT sharder_aggregates_pkey PRIMARY KEY (id, round);

--
-- Name: sharders sharders_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY sharders
    ADD CONSTRAINT sharders_pkey PRIMARY KEY (id);


--
-- Name: snapshots snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY snapshots
    ADD CONSTRAINT snapshots_pkey PRIMARY KEY (round);

--
-- Name: transaction_errors transaction_errors_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY transaction_errors
    ADD CONSTRAINT transaction_errors_pkey PRIMARY KEY (id);


--
-- Name: transactions transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id, round);

--
-- Name: user_aggregates user_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY user_aggregates
    ADD CONSTRAINT user_aggregates_pkey PRIMARY KEY (user_id, round);

--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: validator_aggregates validator_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY validator_aggregates
    ADD CONSTRAINT validator_aggregates_pkey PRIMARY KEY (id, round);

--
-- Name: validators validators_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY validators
    ADD CONSTRAINT validators_pkey PRIMARY KEY (id);


--
-- Name: write_markers write_markers_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY write_markers
    ADD CONSTRAINT write_markers_pkey PRIMARY KEY (id);


--
-- Name: idx_authorizer_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_authorizer_aggregate ON ONLY authorizer_aggregates USING btree (authorizer_id, round);

--
-- Name: idx_authorizer_aggregates_repl; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_authorizer_aggregates_repl ON ONLY authorizer_aggregates USING btree (round, authorizer_id);

--
-- Name: authorizer_snapshots_authorizer_id_idx; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX authorizer_snapshots_authorizer_id_idx ON authorizer_snapshots USING btree (authorizer_id);


--
-- Name: idx_blobber_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_blobber_aggregate ON ONLY blobber_aggregates USING btree (round, blobber_id);

--
-- Name: idx_blobber_aggregates_repl; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_blobber_aggregates_repl ON ONLY blobber_aggregates USING btree (round, blobber_id);

--
-- Name: blobber_snapshots_blobber_id_idx; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX blobber_snapshots_blobber_id_idx ON blobber_snapshots USING btree (blobber_id);


--
-- Name: idx_bhash; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_bhash ON ONLY blocks USING btree (hash, round);

--
-- Name: idx_bround; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_bround ON ONLY blocks USING btree (round);

--
-- Name: idx_alloc_blob; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_alloc_blob ON allocation_blobber_terms USING btree (alloc_id, blobber_id);


--
-- Name: idx_allocation_blobber_terms_deleted_at; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_allocation_blobber_terms_deleted_at ON allocation_blobber_terms USING btree (deleted_at);


--
-- Name: idx_allocations_allocation_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_allocations_allocation_id ON allocations USING btree (allocation_id);


--
-- Name: idx_allocations_deleted_at; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_allocations_deleted_at ON allocations USING btree (deleted_at);


--
-- Name: idx_aowner; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_aowner ON allocations USING btree (owner);


--
-- Name: idx_blobbers_base_url; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_blobbers_base_url ON blobbers USING btree (base_url);


--
-- Name: idx_cchallenge_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_cchallenge_id ON challenges USING btree (challenge_id);


--
-- Name: idx_challenge_pools_allocation_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_challenge_pools_allocation_id ON challenge_pools USING btree (allocation_id);


--
-- Name: idx_copen_challenge; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_copen_challenge ON challenges USING btree (created_at, blobber_id, responded);


--
-- Name: idx_ddel_active; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_ddel_active ON delegate_pools USING btree (provider_type, provider_id, delegate_id, status, pool_id);


--
-- Name: idx_dp_total_staked; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_dp_total_staked ON delegate_pools USING btree (delegate_id, status);


--
-- Name: idx_dprov_active; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_dprov_active ON delegate_pools USING btree (provider_id, provider_type, status);


--
-- Name: idx_miner_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_miner_aggregate ON ONLY miner_aggregates USING btree (miner_id, round);


--
-- Name: idx_miner_aggregates_repl; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_miner_aggregates_repl ON ONLY miner_aggregates USING btree (round, miner_id);


--
-- Name: idx_miner_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_miner_creation_round ON miners USING btree (creation_round);


--
-- Name: idx_provider_rewards_provider_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_provider_rewards_provider_id ON provider_rewards USING btree (provider_id);


--
-- Name: idx_provider_status; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_provider_status ON delegate_pools USING btree (provider_id, status);


--
-- Name: idx_ralloc_block; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_ralloc_block ON read_markers USING btree (allocation_id, block_number);


--
-- Name: idx_rauth_alloc; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_rauth_alloc ON read_markers USING btree (auth_ticket, allocation_id);


--
-- Name: idx_read_markers_transaction_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_read_markers_transaction_id ON read_markers USING btree (transaction_id);


--
-- Name: idx_sharder_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_sharder_aggregate ON ONLY sharder_aggregates USING btree (sharder_id, round);


--
-- Name: idx_sharder_aggregates_repl; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_sharder_aggregates_repl ON ONLY sharder_aggregates USING btree (round, sharder_id);


--
-- Name: idx_sharder_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_sharder_creation_round ON sharders USING btree (creation_round);


--
-- Name: idx_tblock_hash; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_tblock_hash ON ONLY transactions USING btree (block_hash);


--
-- Name: idx_tclient_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_tclient_id ON ONLY transactions USING btree (client_id);


--
-- Name: idx_thash; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_thash ON ONLY transactions USING btree (hash, round);


--
-- Name: idx_tround; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_tround ON ONLY transactions USING btree (round);


--
-- Name: idx_tround_thash; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_tround_thash ON ONLY transactions USING btree (round, hash);


--
-- Name: idx_tto_client_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_tto_client_id ON ONLY transactions USING btree (to_client_id);


--
-- Name: idx_user; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_user ON read_pools USING btree (user_id);


--
-- Name: idx_user_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_user_aggregate ON ONLY user_aggregates USING btree (round, user_id);


--
-- Name: idx_user_snapshots; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_user_snapshots ON user_snapshots USING btree (user_id);


--
-- Name: idx_users_user_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_users_user_id ON users USING btree (user_id);


--
-- Name: idx_validator_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_validator_aggregate ON ONLY validator_aggregates USING btree (validator_id, round);


--
-- Name: idx_validator_aggregates_repl; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_validator_aggregates_repl ON ONLY validator_aggregates USING btree (round, validator_id);


--
-- Name: idx_walloc_block; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_walloc_block ON write_markers USING btree (allocation_id, block_number);


--
-- Name: idx_wblocknum; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_wblocknum ON write_markers USING btree (block_number);


--
-- Name: idx_write_markers_transaction_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_write_markers_transaction_id ON write_markers USING btree (transaction_id);

--
-- Name: miner_snapshots_miner_id_idx; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX miner_snapshots_miner_id_idx ON miner_snapshots USING btree (miner_id);


--
-- Name: ppp; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX ppp ON delegate_pools USING btree (pool_id, provider_type, provider_id);

--
-- Name: sharder_snapshots_sharder_id_idx; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX sharder_snapshots_sharder_id_idx ON sharder_snapshots USING btree (sharder_id);

--
-- Name: validator_snapshots_validator_id_idx; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX validator_snapshots_validator_id_idx ON validator_snapshots USING btree (validator_id);

--
-- Name: allocation_blobber_terms fk_allocations_terms; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY allocation_blobber_terms
    ADD CONSTRAINT fk_allocations_terms FOREIGN KEY (alloc_id) REFERENCES allocations(id);


--
-- Name: allocation_blobber_terms fk_allocations_terms_blobber; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY allocation_blobber_terms
    ADD CONSTRAINT fk_allocations_terms_blobber FOREIGN KEY (blobber_id) REFERENCES blobbers(id);


--
-- Name: read_markers fk_blobbers_read_markers; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY read_markers
    ADD CONSTRAINT fk_blobbers_read_markers FOREIGN KEY (blobber_id) REFERENCES blobbers(id);


--
-- Name: write_markers fk_blobbers_write_markers; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY write_markers
    ADD CONSTRAINT fk_blobbers_write_markers FOREIGN KEY (blobber_id) REFERENCES blobbers(id);


--
-- Name: read_markers fk_read_markers_allocation; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY read_markers
    ADD CONSTRAINT fk_read_markers_allocation FOREIGN KEY (allocation_id) REFERENCES allocations(allocation_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: write_markers fk_write_markers_allocation; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY write_markers
    ADD CONSTRAINT fk_write_markers_allocation FOREIGN KEY (allocation_id) REFERENCES allocations(allocation_id) ON UPDATE CASCADE ON DELETE CASCADE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
