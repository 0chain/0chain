-- +goose Up
-- +goose StatementBegin
--
-- Name: read_markers fk_read_markers_owner; Type: FK CONSTRAINT; Schema: public; Owner: zchain_user
--

ALTER TABLE ONLY public.read_markers
    DROP CONSTRAINT fk_read_markers_user;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- +goose StatementEnd
