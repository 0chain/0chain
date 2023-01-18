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

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: allocation_blobber_terms; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.allocation_blobber_terms (
                                                 id bigint NOT NULL,
                                                 created_at timestamp with time zone,
                                                 updated_at timestamp with time zone,
                                                 deleted_at timestamp with time zone,
                                                 allocation_id text NOT NULL,
                                                 blobber_id text NOT NULL,
                                                 read_price bigint,
                                                 write_price bigint,
                                                 min_lock_demand numeric,
                                                 max_offer_duration bigint
);


ALTER TABLE public.allocation_blobber_terms OWNER TO zchain_user;

--
-- Name: allocation_blobber_terms_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.allocation_blobber_terms_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.allocation_blobber_terms_id_seq OWNER TO zchain_user;

--
-- Name: allocation_blobber_terms_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.allocation_blobber_terms_id_seq OWNED BY public.allocation_blobber_terms.id;


--
-- Name: allocations; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.allocations (
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
                                    is_immutable boolean,
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
                                    write_pool bigint
);


ALTER TABLE public.allocations OWNER TO zchain_user;

--
-- Name: allocations_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.allocations_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.allocations_id_seq OWNER TO zchain_user;

--
-- Name: allocations_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.allocations_id_seq OWNED BY public.allocations.id;


--
-- Name: authorizer_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.authorizer_aggregates (
                                              id bigint NOT NULL,
                                              created_at timestamp with time zone,
                                              authorizer_id text,
                                              round bigint,
                                              bucket_id bigint,
                                              fee bigint,
                                              unstake_total bigint,
                                              total_stake bigint,
                                              total_rewards bigint,
                                              service_charge numeric
);


ALTER TABLE public.authorizer_aggregates OWNER TO zchain_user;

--
-- Name: authorizer_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.authorizer_aggregates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.authorizer_aggregates_id_seq OWNER TO zchain_user;

--
-- Name: authorizer_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.authorizer_aggregates_id_seq OWNED BY public.authorizer_aggregates.id;


--
-- Name: authorizer_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.authorizer_snapshots (
                                             authorizer_id text,
                                             round bigint,
                                             fee bigint,
                                             unstake_total bigint,
                                             total_stake bigint,
                                             total_rewards bigint,
                                             service_charge numeric,
                                             creation_round bigint
);


ALTER TABLE public.authorizer_snapshots OWNER TO zchain_user;

--
-- Name: authorizers; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.authorizers (
                                    id text NOT NULL,
                                    created_at timestamp with time zone,
                                    updated_at timestamp with time zone,
                                    bucket_id bigint,
                                    delegate_wallet text,
                                    min_stake bigint,
                                    max_stake bigint,
                                    num_delegates bigint,
                                    service_charge numeric,
                                    unstake_total bigint,
                                    total_stake bigint,
                                    downtime bigint,
                                    last_health_check bigint,
                                    url text,
                                    fee bigint,
                                    latitude numeric,
                                    longitude numeric,
                                    creation_round bigint
);


ALTER TABLE public.authorizers OWNER TO zchain_user;

--
-- Name: blobber_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.blobber_aggregates (
                                           id bigint NOT NULL,
                                           created_at timestamp with time zone,
                                           blobber_id text,
                                           round bigint,
                                           bucket_id bigint,
                                           write_price bigint,
                                           capacity bigint,
                                           allocated bigint,
                                           saved_data bigint,
                                           read_data bigint,
                                           offers_total bigint,
                                           unstake_total bigint,
                                           total_stake bigint,
                                           total_service_charge bigint,
                                           total_rewards bigint,
                                           challenges_passed bigint,
                                           challenges_completed bigint,
                                           open_challenges bigint,
                                           inactive_rounds bigint,
                                           rank_metric numeric,
                                           downtime bigint
);


ALTER TABLE public.blobber_aggregates OWNER TO zchain_user;

--
-- Name: blobber_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.blobber_aggregates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.blobber_aggregates_id_seq OWNER TO zchain_user;

--
-- Name: blobber_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.blobber_aggregates_id_seq OWNED BY public.blobber_aggregates.id;


--
-- Name: blobber_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.blobber_snapshots (
                                          blobber_id text,
                                          write_price bigint,
                                          capacity bigint,
                                          allocated bigint,
                                          saved_data bigint,
                                          read_data bigint,
                                          offers_total bigint,
                                          unstake_total bigint,
                                          total_service_charge bigint,
                                          total_rewards bigint,
                                          total_stake bigint,
                                          challenges_passed bigint,
                                          challenges_completed bigint,
                                          open_challenges bigint,
                                          creation_round bigint,
                                          rank_metric numeric
);


ALTER TABLE public.blobber_snapshots OWNER TO zchain_user;

--
-- Name: blobbers; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.blobbers (
                                 id text NOT NULL,
                                 created_at timestamp with time zone,
                                 updated_at timestamp with time zone,
                                 bucket_id bigint,
                                 delegate_wallet text,
                                 min_stake bigint,
                                 max_stake bigint,
                                 num_delegates bigint,
                                 service_charge numeric,
                                 unstake_total bigint,
                                 total_stake bigint,
                                 downtime bigint,
                                 last_health_check bigint,
                                 base_url text,
                                 latitude numeric,
                                 longitude numeric,
                                 read_price bigint,
                                 write_price bigint,
                                 min_lock_demand numeric,
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
                                 creation_round bigint
);


ALTER TABLE public.blobbers OWNER TO zchain_user;

--
-- Name: blocks; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.blocks (
                               id bigint NOT NULL,
                               created_at timestamp with time zone,
                               updated_at timestamp with time zone,
                               hash text,
                               version text,
                               creation_date bigint,
                               round bigint,
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
                               round_timeout_count bigint,
                               is_finalised boolean
);


ALTER TABLE public.blocks OWNER TO zchain_user;

--
-- Name: blocks_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.blocks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.blocks_id_seq OWNER TO zchain_user;

--
-- Name: blocks_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.blocks_id_seq OWNED BY public.blocks.id;


--
-- Name: challenge_pools; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.challenge_pools (
                                        id text NOT NULL,
                                        allocation_id text,
                                        balance bigint,
                                        start_time bigint,
                                        expiration bigint,
                                        finalized boolean
);


ALTER TABLE public.challenge_pools OWNER TO zchain_user;

--
-- Name: challenges; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.challenges (
                                   id bigint NOT NULL,
                                   created_at bigint,
                                   updated_at timestamp with time zone,
                                   deleted_at timestamp with time zone,
                                   challenge_id text,
                                   allocation_id text,
                                   blobber_id text,
                                   validators_id text,
                                   seed bigint,
                                   allocation_root text,
                                   responded boolean,
                                   passed boolean,
                                   round_responded bigint
);


ALTER TABLE public.challenges OWNER TO zchain_user;

--
-- Name: challenges_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.challenges_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.challenges_id_seq OWNER TO zchain_user;

--
-- Name: challenges_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.challenges_id_seq OWNED BY public.challenges.id;


--
-- Name: curators; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.curators (
                                 id bigint NOT NULL,
                                 created_at timestamp with time zone,
                                 updated_at timestamp with time zone,
                                 deleted_at timestamp with time zone,
                                 curator_id text,
                                 allocation_id text
);


ALTER TABLE public.curators OWNER TO zchain_user;

--
-- Name: curators_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.curators_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.curators_id_seq OWNER TO zchain_user;

--
-- Name: curators_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.curators_id_seq OWNED BY public.curators.id;


--
-- Name: delegate_pools; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.delegate_pools (
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
                                       round_created bigint
);


ALTER TABLE public.delegate_pools OWNER TO zchain_user;

--
-- Name: delegate_pools_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.delegate_pools_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.delegate_pools_id_seq OWNER TO zchain_user;

--
-- Name: delegate_pools_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.delegate_pools_id_seq OWNED BY public.delegate_pools.id;


--
-- Name: errors; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.errors (
                               id bigint NOT NULL,
                               created_at timestamp with time zone,
                               transaction_id text,
                               error text
);


ALTER TABLE public.errors OWNER TO zchain_user;

--
-- Name: errors_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.errors_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.errors_id_seq OWNER TO zchain_user;

--
-- Name: errors_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.errors_id_seq OWNED BY public.errors.id;


--
-- Name: events; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.events (
                               id bigint NOT NULL,
                               created_at timestamp with time zone,
                               block_number bigint,
                               tx_hash text,
                               type bigint,
                               tag bigint,
                               index text
);


ALTER TABLE public.events OWNER TO zchain_user;

--
-- Name: events_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.events_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.events_id_seq OWNER TO zchain_user;

--
-- Name: events_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.events_id_seq OWNED BY public.events.id;


--
-- Name: miner_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.miner_aggregates (
                                         id bigint NOT NULL,
                                         created_at timestamp with time zone,
                                         miner_id text,
                                         round bigint,
                                         bucket_id bigint,
                                         fees bigint,
                                         unstake_total bigint,
                                         total_stake bigint,
                                         total_rewards bigint,
                                         service_charge numeric
);


ALTER TABLE public.miner_aggregates OWNER TO zchain_user;

--
-- Name: miner_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.miner_aggregates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.miner_aggregates_id_seq OWNER TO zchain_user;

--
-- Name: miner_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.miner_aggregates_id_seq OWNED BY public.miner_aggregates.id;


--
-- Name: miner_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.miner_snapshots (
                                        miner_id text,
                                        round bigint,
                                        fees bigint,
                                        unstake_total bigint,
                                        total_stake bigint,
                                        total_rewards bigint,
                                        service_charge numeric,
                                        creation_round bigint
);


ALTER TABLE public.miner_snapshots OWNER TO zchain_user;

--
-- Name: miners; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.miners (
                               id text NOT NULL,
                               created_at timestamp with time zone,
                               updated_at timestamp with time zone,
                               bucket_id bigint,
                               delegate_wallet text,
                               min_stake bigint,
                               max_stake bigint,
                               num_delegates bigint,
                               service_charge numeric,
                               unstake_total bigint,
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
                               longitude numeric,
                               latitude numeric,
                               creation_round bigint
);


ALTER TABLE public.miners OWNER TO zchain_user;

--
-- Name: provider_rewards; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.provider_rewards (
                                         id bigint NOT NULL,
                                         created_at timestamp with time zone,
                                         updated_at timestamp with time zone,
                                         provider_id text,
                                         rewards bigint,
                                         total_rewards bigint
);


ALTER TABLE public.provider_rewards OWNER TO zchain_user;

--
-- Name: provider_rewards_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.provider_rewards_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.provider_rewards_id_seq OWNER TO zchain_user;

--
-- Name: provider_rewards_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.provider_rewards_id_seq OWNED BY public.provider_rewards.id;


--
-- Name: read_markers; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.read_markers (
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


ALTER TABLE public.read_markers OWNER TO zchain_user;

--
-- Name: read_markers_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.read_markers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.read_markers_id_seq OWNER TO zchain_user;

--
-- Name: read_markers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.read_markers_id_seq OWNED BY public.read_markers.id;


--
-- Name: reward_delegates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.reward_delegates (
                                         id bigint NOT NULL,
                                         created_at timestamp with time zone,
                                         updated_at timestamp with time zone,
                                         amount bigint,
                                         block_number bigint,
                                         pool_id text,
                                         reward_type bigint
);


ALTER TABLE public.reward_delegates OWNER TO zchain_user;

--
-- Name: reward_delegates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.reward_delegates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.reward_delegates_id_seq OWNER TO zchain_user;

--
-- Name: reward_delegates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.reward_delegates_id_seq OWNED BY public.reward_delegates.id;


--
-- Name: reward_mints; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.reward_mints (
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


ALTER TABLE public.reward_mints OWNER TO zchain_user;

--
-- Name: reward_mints_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.reward_mints_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.reward_mints_id_seq OWNER TO zchain_user;

--
-- Name: reward_mints_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.reward_mints_id_seq OWNED BY public.reward_mints.id;


--
-- Name: reward_providers; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.reward_providers (
                                         id bigint NOT NULL,
                                         created_at timestamp with time zone,
                                         updated_at timestamp with time zone,
                                         amount bigint,
                                         block_number bigint,
                                         provider_id text,
                                         reward_type bigint
);


ALTER TABLE public.reward_providers OWNER TO zchain_user;

--
-- Name: reward_providers_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.reward_providers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.reward_providers_id_seq OWNER TO zchain_user;

--
-- Name: reward_providers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.reward_providers_id_seq OWNED BY public.reward_providers.id;


--
-- Name: sharder_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.sharder_aggregates (
                                           id bigint NOT NULL,
                                           created_at timestamp with time zone,
                                           sharder_id text,
                                           round bigint,
                                           bucket_id bigint,
                                           fees bigint,
                                           unstake_total bigint,
                                           total_stake bigint,
                                           total_rewards bigint,
                                           service_charge numeric
);


ALTER TABLE public.sharder_aggregates OWNER TO zchain_user;

--
-- Name: sharder_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.sharder_aggregates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.sharder_aggregates_id_seq OWNER TO zchain_user;

--
-- Name: sharder_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.sharder_aggregates_id_seq OWNED BY public.sharder_aggregates.id;


--
-- Name: sharder_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.sharder_snapshots (
                                          sharder_id text,
                                          round bigint,
                                          fees bigint,
                                          unstake_total bigint,
                                          total_stake bigint,
                                          total_rewards bigint,
                                          service_charge numeric,
                                          creation_round bigint
);


ALTER TABLE public.sharder_snapshots OWNER TO zchain_user;

--
-- Name: sharders; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.sharders (
                                 id text NOT NULL,
                                 created_at timestamp with time zone,
                                 updated_at timestamp with time zone,
                                 bucket_id bigint,
                                 delegate_wallet text,
                                 min_stake bigint,
                                 max_stake bigint,
                                 num_delegates bigint,
                                 service_charge numeric,
                                 unstake_total bigint,
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
                                 longitude numeric,
                                 latitude numeric,
                                 creation_round bigint
);


ALTER TABLE public.sharders OWNER TO zchain_user;

--
-- Name: snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.snapshots (
                                  round bigint NOT NULL,
                                  total_mint bigint,
                                  total_challenge_pools bigint,
                                  active_allocated_delta bigint,
                                  zcn_supply bigint,
                                  total_value_locked bigint,
                                  client_locks bigint,
                                  mined_total bigint,
                                  average_write_price bigint,
                                  total_staked bigint,
                                  successful_challenges bigint,
                                  total_challenges bigint,
                                  allocated_storage bigint,
                                  max_capacity_storage bigint,
                                  staked_storage bigint,
                                  used_storage bigint,
                                  transactions_count bigint,
                                  unique_addresses bigint,
                                  block_count bigint,
                                  average_txn_fee bigint,
                                  created_at bigint
);


ALTER TABLE public.snapshots OWNER TO zchain_user;

--
-- Name: transactions; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.transactions (
                                     id bigint NOT NULL,
                                     created_at timestamp with time zone,
                                     hash text,
                                     block_hash text,
                                     round bigint,
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
);


ALTER TABLE public.transactions OWNER TO zchain_user;

--
-- Name: transactions_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.transactions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.transactions_id_seq OWNER TO zchain_user;

--
-- Name: transactions_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.transactions_id_seq OWNED BY public.transactions.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.users (
                              id bigint NOT NULL,
                              created_at timestamp with time zone,
                              updated_at timestamp with time zone,
                              user_id text,
                              txn_hash text,
                              balance bigint,
                              change bigint,
                              round bigint,
                              nonce bigint
);


ALTER TABLE public.users OWNER TO zchain_user;

--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.users_id_seq OWNER TO zchain_user;

--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: validator_aggregates; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.validator_aggregates (
                                             id bigint NOT NULL,
                                             created_at timestamp with time zone,
                                             validator_id text,
                                             round bigint,
                                             bucket_id bigint,
                                             unstake_total bigint,
                                             total_stake bigint,
                                             total_rewards bigint,
                                             service_charge numeric
);


ALTER TABLE public.validator_aggregates OWNER TO zchain_user;

--
-- Name: validator_aggregates_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.validator_aggregates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.validator_aggregates_id_seq OWNER TO zchain_user;

--
-- Name: validator_aggregates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.validator_aggregates_id_seq OWNED BY public.validator_aggregates.id;


--
-- Name: validator_snapshots; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.validator_snapshots (
                                            validator_id text,
                                            unstake_total bigint,
                                            total_stake bigint,
                                            total_rewards bigint,
                                            service_charge numeric,
                                            creation_round bigint
);


ALTER TABLE public.validator_snapshots OWNER TO zchain_user;

--
-- Name: validators; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.validators (
                                   id text NOT NULL,
                                   created_at timestamp with time zone,
                                   updated_at timestamp with time zone,
                                   bucket_id bigint,
                                   delegate_wallet text,
                                   min_stake bigint,
                                   max_stake bigint,
                                   num_delegates bigint,
                                   service_charge numeric,
                                   unstake_total bigint,
                                   total_stake bigint,
                                   downtime bigint,
                                   last_health_check bigint,
                                   base_url text,
                                   public_key text,
                                   creation_round bigint
);


ALTER TABLE public.validators OWNER TO zchain_user;

--
-- Name: write_markers; Type: TABLE; Schema: public; Owner: zchain_user
--

CREATE TABLE public.write_markers (
                                      id bigint NOT NULL,
                                      created_at timestamp with time zone,
                                      updated_at timestamp with time zone,
                                      client_id text,
                                      blobber_id text,
                                      allocation_id text,
                                      transaction_id text,
                                      allocation_root text,
                                      previous_allocation_root text,
                                      size bigint,
                                      "timestamp" bigint,
                                      signature text,
                                      block_number bigint,
                                      lookup_hash text,
                                      name text,
                                      content_hash text,
                                      operation text
);


ALTER TABLE public.write_markers OWNER TO zchain_user;

--
-- Name: write_markers_id_seq; Type: SEQUENCE; Schema: public; Owner: zchain_user
--

CREATE SEQUENCE public.write_markers_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


ALTER TABLE public.write_markers_id_seq OWNER TO zchain_user;

--
-- Name: write_markers_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: zchain_user
--

ALTER SEQUENCE public.write_markers_id_seq OWNED BY public.write_markers.id;


--
-- Name: allocation_blobber_terms id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.allocation_blobber_terms ALTER COLUMN id SET DEFAULT nextval('public.allocation_blobber_terms_id_seq'::regclass);


--
-- Name: allocations id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.allocations ALTER COLUMN id SET DEFAULT nextval('public.allocations_id_seq'::regclass);


--
-- Name: authorizer_aggregates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.authorizer_aggregates ALTER COLUMN id SET DEFAULT nextval('public.authorizer_aggregates_id_seq'::regclass);


--
-- Name: blobber_aggregates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.blobber_aggregates ALTER COLUMN id SET DEFAULT nextval('public.blobber_aggregates_id_seq'::regclass);


--
-- Name: blocks id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.blocks ALTER COLUMN id SET DEFAULT nextval('public.blocks_id_seq'::regclass);


--
-- Name: challenges id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.challenges ALTER COLUMN id SET DEFAULT nextval('public.challenges_id_seq'::regclass);


--
-- Name: curators id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.curators ALTER COLUMN id SET DEFAULT nextval('public.curators_id_seq'::regclass);


--
-- Name: delegate_pools id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.delegate_pools ALTER COLUMN id SET DEFAULT nextval('public.delegate_pools_id_seq'::regclass);


--
-- Name: errors id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.errors ALTER COLUMN id SET DEFAULT nextval('public.errors_id_seq'::regclass);


--
-- Name: events id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.events ALTER COLUMN id SET DEFAULT nextval('public.events_id_seq'::regclass);


--
-- Name: miner_aggregates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.miner_aggregates ALTER COLUMN id SET DEFAULT nextval('public.miner_aggregates_id_seq'::regclass);


--
-- Name: provider_rewards id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.provider_rewards ALTER COLUMN id SET DEFAULT nextval('public.provider_rewards_id_seq'::regclass);


--
-- Name: read_markers id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.read_markers ALTER COLUMN id SET DEFAULT nextval('public.read_markers_id_seq'::regclass);


--
-- Name: reward_delegates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.reward_delegates ALTER COLUMN id SET DEFAULT nextval('public.reward_delegates_id_seq'::regclass);


--
-- Name: reward_mints id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.reward_mints ALTER COLUMN id SET DEFAULT nextval('public.reward_mints_id_seq'::regclass);


--
-- Name: reward_providers id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.reward_providers ALTER COLUMN id SET DEFAULT nextval('public.reward_providers_id_seq'::regclass);


--
-- Name: sharder_aggregates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.sharder_aggregates ALTER COLUMN id SET DEFAULT nextval('public.sharder_aggregates_id_seq'::regclass);


--
-- Name: transactions id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.transactions ALTER COLUMN id SET DEFAULT nextval('public.transactions_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: validator_aggregates id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.validator_aggregates ALTER COLUMN id SET DEFAULT nextval('public.validator_aggregates_id_seq'::regclass);


--
-- Name: write_markers id; Type: DEFAULT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.write_markers ALTER COLUMN id SET DEFAULT nextval('public.write_markers_id_seq'::regclass);


--
-- Name: allocation_blobber_terms allocation_blobber_terms_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.allocation_blobber_terms
    ADD CONSTRAINT allocation_blobber_terms_pkey PRIMARY KEY (id);


--
-- Name: allocations allocations_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.allocations
    ADD CONSTRAINT allocations_pkey PRIMARY KEY (id);


--
-- Name: authorizer_aggregates authorizer_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.authorizer_aggregates
    ADD CONSTRAINT authorizer_aggregates_pkey PRIMARY KEY (id);


--
-- Name: authorizers authorizers_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.authorizers
    ADD CONSTRAINT authorizers_pkey PRIMARY KEY (id);


--
-- Name: blobber_aggregates blobber_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.blobber_aggregates
    ADD CONSTRAINT blobber_aggregates_pkey PRIMARY KEY (id);


--
-- Name: blobbers blobbers_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.blobbers
    ADD CONSTRAINT blobbers_pkey PRIMARY KEY (id);


--
-- Name: blocks blocks_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.blocks
    ADD CONSTRAINT blocks_pkey PRIMARY KEY (id);


--
-- Name: challenge_pools challenge_pools_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.challenge_pools
    ADD CONSTRAINT challenge_pools_pkey PRIMARY KEY (id);


--
-- Name: challenges challenges_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.challenges
    ADD CONSTRAINT challenges_pkey PRIMARY KEY (id);


--
-- Name: curators curators_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.curators
    ADD CONSTRAINT curators_pkey PRIMARY KEY (id);


--
-- Name: delegate_pools delegate_pools_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.delegate_pools
    ADD CONSTRAINT delegate_pools_pkey PRIMARY KEY (id);


--
-- Name: errors errors_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.errors
    ADD CONSTRAINT errors_pkey PRIMARY KEY (id);


--
-- Name: events events_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.events
    ADD CONSTRAINT events_pkey PRIMARY KEY (id);


--
-- Name: miner_aggregates miner_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.miner_aggregates
    ADD CONSTRAINT miner_aggregates_pkey PRIMARY KEY (id);


--
-- Name: miners miners_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.miners
    ADD CONSTRAINT miners_pkey PRIMARY KEY (id);


--
-- Name: provider_rewards provider_rewards_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.provider_rewards
    ADD CONSTRAINT provider_rewards_pkey PRIMARY KEY (id);


--
-- Name: read_markers read_markers_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.read_markers
    ADD CONSTRAINT read_markers_pkey PRIMARY KEY (id);


--
-- Name: reward_delegates reward_delegates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.reward_delegates
    ADD CONSTRAINT reward_delegates_pkey PRIMARY KEY (id);


--
-- Name: reward_mints reward_mints_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.reward_mints
    ADD CONSTRAINT reward_mints_pkey PRIMARY KEY (id);


--
-- Name: reward_providers reward_providers_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.reward_providers
    ADD CONSTRAINT reward_providers_pkey PRIMARY KEY (id);


--
-- Name: sharder_aggregates sharder_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.sharder_aggregates
    ADD CONSTRAINT sharder_aggregates_pkey PRIMARY KEY (id);


--
-- Name: sharders sharders_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.sharders
    ADD CONSTRAINT sharders_pkey PRIMARY KEY (id);


--
-- Name: snapshots snapshots_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.snapshots
    ADD CONSTRAINT snapshots_pkey PRIMARY KEY (round);


--
-- Name: transactions transactions_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.transactions
    ADD CONSTRAINT transactions_pkey PRIMARY KEY (id);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: validator_aggregates validator_aggregates_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.validator_aggregates
    ADD CONSTRAINT validator_aggregates_pkey PRIMARY KEY (id);


--
-- Name: validators validators_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.validators
    ADD CONSTRAINT validators_pkey PRIMARY KEY (id);


--
-- Name: write_markers write_markers_pkey; Type: CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.write_markers
    ADD CONSTRAINT write_markers_pkey PRIMARY KEY (id);


--
-- Name: idx_alloc_blob; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_alloc_blob ON public.allocation_blobber_terms USING btree (allocation_id, blobber_id);


--
-- Name: idx_allocation_blobber_terms_deleted_at; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_allocation_blobber_terms_deleted_at ON public.allocation_blobber_terms USING btree (deleted_at);


--
-- Name: idx_allocations_allocation_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_allocations_allocation_id ON public.allocations USING btree (allocation_id);


--
-- Name: idx_allocations_deleted_at; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_allocations_deleted_at ON public.allocations USING btree (deleted_at);


--
-- Name: idx_aowner; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_aowner ON public.allocations USING btree (owner);


--
-- Name: idx_astart_time; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_astart_time ON public.allocations USING btree (start_time);


--
-- Name: idx_authorizer_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_authorizer_aggregate ON public.authorizer_aggregates USING btree (authorizer_id, round);


--
-- Name: idx_authorizer_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_authorizer_creation_round ON public.authorizers USING btree (creation_round);


--
-- Name: idx_authorizer_snapshots_authorizer_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_authorizer_snapshots_authorizer_id ON public.authorizer_snapshots USING btree (authorizer_id);


--
-- Name: idx_authorizer_snapshots_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_authorizer_snapshots_creation_round ON public.authorizer_snapshots USING btree (creation_round);


--
-- Name: idx_ba_rankmetric; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_ba_rankmetric ON public.blobber_aggregates USING btree (rank_metric);


--
-- Name: idx_bcreation_date; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_bcreation_date ON public.blocks USING btree (creation_date);


--
-- Name: idx_bhash; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_bhash ON public.blocks USING btree (hash);


--
-- Name: idx_blobber_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_blobber_aggregate ON public.blobber_aggregates USING btree (round, blobber_id);


--
-- Name: idx_blobber_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_blobber_creation_round ON public.blobbers USING btree (creation_round);


--
-- Name: idx_blobber_snapshots_blobber_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_blobber_snapshots_blobber_id ON public.blobber_snapshots USING btree (blobber_id);


--
-- Name: idx_blobber_snapshots_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_blobber_snapshots_creation_round ON public.blobber_snapshots USING btree (creation_round);


--
-- Name: idx_blobbers_base_url; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_blobbers_base_url ON public.blobbers USING btree (base_url);


--
-- Name: idx_blobbers_rank_metric; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_blobbers_rank_metric ON public.blobbers USING btree (rank_metric);


--
-- Name: idx_bround; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_bround ON public.blocks USING btree (round);


--
-- Name: idx_cchallenge_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_cchallenge_id ON public.challenges USING btree (challenge_id);


--
-- Name: idx_challenge_pools_allocation_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_challenge_pools_allocation_id ON public.challenge_pools USING btree (allocation_id);


--
-- Name: idx_challenges_deleted_at; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_challenges_deleted_at ON public.challenges USING btree (deleted_at);


--
-- Name: idx_challenges_round_responded; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_challenges_round_responded ON public.challenges USING btree (round_responded);


--
-- Name: idx_copen_challenge; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_copen_challenge ON public.challenges USING btree (created_at, blobber_id, responded);


--
-- Name: idx_curators_curator_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_curators_curator_id ON public.curators USING btree (curator_id);


--
-- Name: idx_curators_deleted_at; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_curators_deleted_at ON public.curators USING btree (deleted_at);


--
-- Name: idx_ddel_active; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_ddel_active ON public.delegate_pools USING btree (provider_type, provider_id, delegate_id, status, pool_id);


--
-- Name: idx_del_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_del_id ON public.delegate_pools USING btree (delegate_id);


--
-- Name: idx_dp_total_staked; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_dp_total_staked ON public.delegate_pools USING btree (delegate_id, status);


--
-- Name: idx_dprov_active; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_dprov_active ON public.delegate_pools USING btree (provider_id, provider_type, status);


--
-- Name: idx_event; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_event ON public.events USING btree (block_number, tx_hash, type, tag, index);


--
-- Name: idx_miner_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_miner_aggregate ON public.miner_aggregates USING btree (miner_id, round);


--
-- Name: idx_miner_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_miner_creation_round ON public.miners USING btree (creation_round);


--
-- Name: idx_miner_snapshots_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_miner_snapshots_creation_round ON public.miner_snapshots USING btree (creation_round);


--
-- Name: idx_miner_snapshots_miner_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_miner_snapshots_miner_id ON public.miner_snapshots USING btree (miner_id);


--
-- Name: idx_provider_rewards_provider_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_provider_rewards_provider_id ON public.provider_rewards USING btree (provider_id);


--
-- Name: idx_ralloc_block; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_ralloc_block ON public.read_markers USING btree (allocation_id, block_number);


--
-- Name: idx_rauth_alloc; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_rauth_alloc ON public.read_markers USING btree (auth_ticket, allocation_id);


--
-- Name: idx_read_markers_transaction_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_read_markers_transaction_id ON public.read_markers USING btree (transaction_id);


--
-- Name: idx_rew_block_prov; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_rew_block_prov ON public.reward_providers USING btree (block_number, provider_id);


--
-- Name: idx_rew_del_prov; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_rew_del_prov ON public.reward_delegates USING btree (block_number, pool_id);


--
-- Name: idx_sharder_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_sharder_aggregate ON public.sharder_aggregates USING btree (sharder_id, round);


--
-- Name: idx_sharder_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_sharder_creation_round ON public.sharders USING btree (creation_round);


--
-- Name: idx_sharder_snapshots_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_sharder_snapshots_creation_round ON public.sharder_snapshots USING btree (creation_round);


--
-- Name: idx_sharder_snapshots_sharder_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_sharder_snapshots_sharder_id ON public.sharder_snapshots USING btree (sharder_id);


--
-- Name: idx_tblock_hash; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_tblock_hash ON public.transactions USING btree (block_hash);


--
-- Name: idx_tclient_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_tclient_id ON public.transactions USING btree (client_id);


--
-- Name: idx_tcreation_date; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_tcreation_date ON public.transactions USING btree (creation_date);


--
-- Name: idx_thash; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_thash ON public.transactions USING btree (hash);


--
-- Name: idx_tto_client_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_tto_client_id ON public.transactions USING btree (to_client_id);


--
-- Name: idx_users_user_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_users_user_id ON public.users USING btree (user_id);


--
-- Name: idx_validator_aggregate; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_validator_aggregate ON public.validator_aggregates USING btree (validator_id, round);


--
-- Name: idx_validator_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_validator_creation_round ON public.validators USING btree (creation_round);


--
-- Name: idx_validator_snapshots_creation_round; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_validator_snapshots_creation_round ON public.validator_snapshots USING btree (creation_round);


--
-- Name: idx_validator_snapshots_validator_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_validator_snapshots_validator_id ON public.validator_snapshots USING btree (validator_id);


--
-- Name: idx_walloc_block; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_walloc_block ON public.write_markers USING btree (allocation_id, block_number);


--
-- Name: idx_walloc_file; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_walloc_file ON public.write_markers USING btree (allocation_id);


--
-- Name: idx_wblocknum; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_wblocknum ON public.write_markers USING btree (block_number);


--
-- Name: idx_wcontent; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_wcontent ON public.write_markers USING btree (content_hash);


--
-- Name: idx_wlookup; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_wlookup ON public.write_markers USING btree (lookup_hash);


--
-- Name: idx_wname; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE INDEX idx_wname ON public.write_markers USING btree (name);


--
-- Name: idx_write_markers_transaction_id; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX idx_write_markers_transaction_id ON public.write_markers USING btree (transaction_id);


--
-- Name: ppp; Type: INDEX; Schema: public; Owner: zchain_user
--

CREATE UNIQUE INDEX ppp ON public.delegate_pools USING btree (pool_id, provider_type, provider_id);


--
-- Name: allocation_blobber_terms fk_allocations_terms; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.allocation_blobber_terms
    ADD CONSTRAINT fk_allocations_terms FOREIGN KEY (allocation_id) REFERENCES public.allocations(allocation_id);


--
-- Name: allocations fk_allocations_user; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.allocations
    ADD CONSTRAINT fk_allocations_user FOREIGN KEY (owner) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: read_markers fk_blobbers_read_markers; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.read_markers
    ADD CONSTRAINT fk_blobbers_read_markers FOREIGN KEY (blobber_id) REFERENCES public.blobbers(id);


--
-- Name: write_markers fk_blobbers_write_markers; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.write_markers
    ADD CONSTRAINT fk_blobbers_write_markers FOREIGN KEY (blobber_id) REFERENCES public.blobbers(id);


--
-- Name: curators fk_curators_allocation; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.curators
    ADD CONSTRAINT fk_curators_allocation FOREIGN KEY (allocation_id) REFERENCES public.allocations(allocation_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: curators fk_curators_user; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.curators
    ADD CONSTRAINT fk_curators_user FOREIGN KEY (curator_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: read_markers fk_read_markers_allocation; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.read_markers
    ADD CONSTRAINT fk_read_markers_allocation FOREIGN KEY (allocation_id) REFERENCES public.allocations(allocation_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: read_markers fk_read_markers_owner; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.read_markers
    ADD CONSTRAINT fk_read_markers_owner FOREIGN KEY (owner_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: read_markers fk_read_markers_user; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.read_markers
    ADD CONSTRAINT fk_read_markers_user FOREIGN KEY (client_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: write_markers fk_write_markers_allocation; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.write_markers
    ADD CONSTRAINT fk_write_markers_allocation FOREIGN KEY (allocation_id) REFERENCES public.allocations(allocation_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- Name: write_markers fk_write_markers_user; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.write_markers
    ADD CONSTRAINT fk_write_markers_user FOREIGN KEY (client_id) REFERENCES public.users(user_id) ON UPDATE CASCADE ON DELETE CASCADE;


--
-- PostgreSQL database dump complete
--
-- +goose StatementEnd

