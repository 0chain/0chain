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


CREATE TABLE public.user_aggregates (
    user_id text,
    round bigint,
    collected_reward bigint,
    total_stake bigint,
    read_pool_total bigint,
    write_pool_total bigint,
    payed_fees bigint,
    created_at timestamp with time zone
) PARTITION BY RANGE (round);

ALTER TABLE public.user_aggregates OWNER TO zchain_user;

CREATE UNIQUE INDEX idx_user_aggregate ON public.user_aggregates USING btree (round, user_id);


ALTER TABLE public.user_aggregates
    ADD CONSTRAINT user_aggregates_pkey PRIMARY KEY (user_id, round);

--
-- Migration complete
--

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TABLE public.user_aggregates;

DROP INDEX idx_user_aggregate;

-- +goose StatementEnd