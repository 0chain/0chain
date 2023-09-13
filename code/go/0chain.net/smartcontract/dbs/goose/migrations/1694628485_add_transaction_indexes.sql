-- +goose Up
-- +goose StatementBegin
CREATE INDEX idx_tround ON transactions USING btree (round);
CREATE INDEX idx_tround_thash ON transactions USING btree (round, hash);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX idx_tround;
DROP INDEX idx_tround_thash;
-- +goose StatementEnd
