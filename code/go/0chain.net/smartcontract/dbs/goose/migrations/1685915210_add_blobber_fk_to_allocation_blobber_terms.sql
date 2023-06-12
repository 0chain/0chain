-- +goose Up
-- +goose StatementBegin
ALTER TABLE allocation_blobber_terms
    ADD CONSTRAINT fk_allocations_terms_blobber FOREIGN KEY (blobber_id) REFERENCES public.blobbers(id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE allocation_blobber_terms
    DROP CONSTRAINT fk_allocations_terms_blobber;
-- +goose StatementEnd
