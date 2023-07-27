-- +goose Up
-- +goose StatementBegin
CREATE UNIQUE INDEX idx_user ON public.read_pools USING btree (user_id);
ALTER TABLE public.read_pools
    ADD CONSTRAINT read_pools_pkey PRIMARY KEY (user_id);
-- +goose StatementEnd