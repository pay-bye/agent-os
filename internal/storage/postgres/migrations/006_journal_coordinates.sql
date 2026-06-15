ALTER TABLE journal_events
  ADD COLUMN coordinate_kind TEXT,
  ADD COLUMN coordinate_key TEXT;

UPDATE journal_events
SET coordinate_kind = 'work_item',
    coordinate_key = work_item_id;

ALTER TABLE journal_events
  ALTER COLUMN coordinate_kind SET NOT NULL,
  ALTER COLUMN coordinate_key SET NOT NULL,
  ADD CONSTRAINT journal_events_coordinate_kind_known
    CHECK (coordinate_kind IN ('work_item', 'node')),
  ADD CONSTRAINT journal_events_coordinate_key_not_empty
    CHECK (length(btrim(coordinate_key)) > 0);

DROP INDEX journal_events_work_item_append_order;

CREATE INDEX journal_events_coordinate_append_order
  ON journal_events (coordinate_kind, coordinate_key, append_index);

ALTER TABLE journal_events
  DROP COLUMN work_item_id;

CREATE TABLE routing_exclusions (
  node_key TEXT PRIMARY KEY CONSTRAINT routing_exclusions_node_key_not_empty CHECK (length(btrim(node_key)) > 0),
  FOREIGN KEY (node_key) REFERENCES nodes(key)
);

INSERT INTO journal_event_kinds (key, schema_key, description) VALUES
  ('x45', NULL, 'Routing exclusion set'),
  ('x46', NULL, 'Routing exclusion cleared')
ON CONFLICT (key) DO NOTHING;
