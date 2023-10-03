-- +goose Up
-- +goose StatementBegin
CREATE INDEX idx_tround ON public.transactions USING btree (round);
CREATE INDEX idx_tround_thash ON public.transactions USING btree (round, hash);
CREATE INDEX idx_provider_status ON public.delegate_pools USING btree (provider_id, status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_tround;
DROP INDEX idx_tround_thash;
DROP INDEX idx_provider_status;
-- +goose StatementEnd
