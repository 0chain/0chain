-- noinspection SqlDialectInspectionForFile

CREATE TABLE IF NOT EXISTS zerochain.magic_block_map (
    id bigint,
    hash text,
    block_round bigint,
    PRIMARY KEY (id)
);
