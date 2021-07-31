CREATE TABLE IF NOT EXISTS zerochain.magic_block_map (
    id text,
    hash text,
    block_round bigint,
    PRIMARY KEY (id)
);
