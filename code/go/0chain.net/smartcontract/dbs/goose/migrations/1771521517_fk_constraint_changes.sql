-- +goose Up
-- +goose StatementBegin

-- Remove FK on allocations.owner → users.user_id to prevent insert failures
-- when allocation events arrive before user events in the same batch
ALTER TABLE allocations DROP CONSTRAINT IF EXISTS fk_allocations_user;

-- Change read_markers FK from CASCADE to SET NULL to preserve historical records
ALTER TABLE read_markers DROP CONSTRAINT IF EXISTS fk_read_markers_allocation;
ALTER TABLE read_markers ADD CONSTRAINT fk_read_markers_allocation
  FOREIGN KEY (allocation_id) REFERENCES allocations(allocation_id)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- Change write_markers FK from CASCADE to SET NULL to preserve historical records
ALTER TABLE write_markers DROP CONSTRAINT IF EXISTS fk_write_markers_allocation;
ALTER TABLE write_markers ADD CONSTRAINT fk_write_markers_allocation
  FOREIGN KEY (allocation_id) REFERENCES allocations(allocation_id)
  ON UPDATE CASCADE ON DELETE SET NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Restore original CASCADE constraints
ALTER TABLE allocations ADD CONSTRAINT fk_allocations_user
  FOREIGN KEY (owner) REFERENCES users(user_id)
  ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE read_markers DROP CONSTRAINT IF EXISTS fk_read_markers_allocation;
ALTER TABLE read_markers ADD CONSTRAINT fk_read_markers_allocation
  FOREIGN KEY (allocation_id) REFERENCES allocations(allocation_id)
  ON UPDATE CASCADE ON DELETE CASCADE;

ALTER TABLE write_markers DROP CONSTRAINT IF EXISTS fk_write_markers_allocation;
ALTER TABLE write_markers ADD CONSTRAINT fk_write_markers_allocation
  FOREIGN KEY (allocation_id) REFERENCES allocations(allocation_id)
  ON UPDATE CASCADE ON DELETE CASCADE;

-- +goose StatementEnd
