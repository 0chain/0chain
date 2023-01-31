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


ALTER TABLE public.users
    ADD bucket_id bigint NOT NULL DEFAULT 0,
    ADD collected_reward bigint,
    ADD total_stake bigint,
    ADD read_pool_total bigint,
    ADD write_pool_total bigint,
    ADD payed_fees bigint
;


CREATE TABLE public.user_snapshots (
    user_id text,
    round bigint,
    collected_reward bigint,
    total_stake bigint,
    read_pool_total bigint,
    write_pool_total bigint,
    payed_fees bigint
);

ALTER TABLE public.user_snapshots OWNER TO zchain_user;


CREATE TABLE public.user_aggregates (
    user_id text,
    round bigint,
    collected_reward bigint,
    total_stake bigint,
    read_pool_total bigint,
    write_pool_total bigint,
    payed_fees bigint,
    created_at timestamp with time zone
);

ALTER TABLE public.user_aggregates OWNER TO zchain_user;


CREATE INDEX idx_user_snapshot_user_id ON public.user_snapshots USING btree (user_id);


CREATE UNIQUE INDEX idx_user_aggregate ON public.user_aggregates USING btree (round, user_id);


--
-- Migration complete
--


-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin


ALTER TABLE public.users
    DROP COLUMN bucket_id, 
    DROP COLUMN collected_reward, 
    DROP COLUMN total_stake, 
    DROP COLUMN read_pool_total,
    DROP COLUMN write_pool_total,
    DROP COLUMN payed_fees
;

DROP TABLE public.user_snapshots, public.user_aggregates;

DROP INDEX idx_user_snapshot_user_id;

DROP INDEX idx_user_aggregate;


-- +goose StatementEnd