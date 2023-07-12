-- +goose Up
-- +goose StatementBegin
CREATE TABLE read_pools (
    id bigint,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,

    user_id text NOT NULL,
    balance bigint
);

ALTER TABLE public.read_pools OWNER TO zchain_user;

CREATE SEQUENCE public.read_pools_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER TABLE public.read_pools_id_seq OWNER TO zchain_user;

ALTER SEQUENCE public.read_pools_id_seq OWNED BY public.read_pools.id;
-- +goose StatementEnd