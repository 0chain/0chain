-- +goose Up
-- +goose StatementBegin
CREATE TABLE readpool (
                                user_id text,
                                balance bigint,
);

CREATE UNIQUE INDEX idx_user_readpools ON readpools USING btree (user_id);
-- +goose StatementEnd