-- +goose Up
-- +goose StatementBegin

CREATE INDEX idx_blobber_aggregates_repl ON public.blobber_aggregates USING btree (round, blobber_id);

CREATE INDEX idx_authorizer_aggregates_repl ON public.authorizer_aggregates USING btree (round, authorizer_id);

CREATE INDEX idx_sharder_aggregates_repl ON public.sharder_aggregates USING btree (round, sharder_id);

CREATE INDEX idx_miner_aggregates_repl ON public.miner_aggregates USING btree (round, miner_id);

CREATE INDEX idx_validator_aggregates_repl ON public.validator_aggregates USING btree (round, validator_id);

-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin

DROP INDEX idx_blobber_aggregates_repl;

DROP INDEX idx_authorizer_aggregates_repl;

DROP INDEX idx_sharder_aggregates_repl;

DROP INDEX idx_validator_aggregates_repl;

DROP INDEX idx_miner_aggregates_repl;

-- +goose StatementEnd