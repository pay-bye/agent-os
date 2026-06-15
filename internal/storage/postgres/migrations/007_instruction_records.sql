ALTER TABLE journal_events
  DROP CONSTRAINT journal_events_coordinate_kind_known,
  ADD CONSTRAINT journal_events_coordinate_kind_known
    CHECK (coordinate_kind IN ('work_item', 'node', 'channel', 'lease'));

CREATE TABLE instruction_records (
  instruction_id TEXT PRIMARY KEY CONSTRAINT instruction_records_id_not_empty CHECK (length(btrim(instruction_id)) > 0),
  kind TEXT NOT NULL CONSTRAINT instruction_records_kind_not_empty CHECK (length(btrim(kind)) > 0),
  request_digest TEXT NOT NULL CONSTRAINT instruction_records_digest_not_empty CHECK (length(btrim(request_digest)) > 0),
  result TEXT CONSTRAINT instruction_records_result_known CHECK (result IS NULL OR result IN ('applied', 'precondition_failed')),
  affected_ids TEXT[] NOT NULL DEFAULT '{}',
  event_ids TEXT[] NOT NULL DEFAULT '{}',
  failed_precondition TEXT CONSTRAINT instruction_records_failed_precondition_not_empty CHECK (failed_precondition IS NULL OR length(btrim(failed_precondition)) > 0),
  recorded_at TIMESTAMPTZ NOT NULL
);

INSERT INTO journal_event_kinds (key, schema_key, description) VALUES
  ('x47', NULL, 'Instruction applied'),
  ('x48', NULL, 'Instruction rejected'),
  ('x49', NULL, 'Work item dropped')
ON CONFLICT (key) DO NOTHING;
