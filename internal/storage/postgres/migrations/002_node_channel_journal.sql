CREATE TABLE nodes (
  key TEXT PRIMARY KEY CONSTRAINT nodes_key_not_empty CHECK (length(btrim(key)) > 0),
  description TEXT NOT NULL CONSTRAINT nodes_description_not_empty CHECK (length(btrim(description)) > 0)
);

CREATE TABLE channels (
  key TEXT PRIMARY KEY CONSTRAINT channels_key_not_empty CHECK (length(btrim(key)) > 0),
  node_key TEXT NOT NULL UNIQUE CONSTRAINT channels_node_key_not_empty CHECK (length(btrim(node_key)) > 0),
  description TEXT NOT NULL CONSTRAINT channels_description_not_empty CHECK (length(btrim(description)) > 0),
  FOREIGN KEY (node_key) REFERENCES nodes(key)
);

CREATE TABLE node_capabilities (
  node_key TEXT NOT NULL CONSTRAINT node_capabilities_node_key_not_empty CHECK (length(btrim(node_key)) > 0),
  need_kind_key TEXT NOT NULL CONSTRAINT node_capabilities_need_kind_key_not_empty CHECK (length(btrim(need_kind_key)) > 0),
  PRIMARY KEY (node_key, need_kind_key),
  FOREIGN KEY (node_key) REFERENCES nodes(key),
  FOREIGN KEY (need_kind_key) REFERENCES need_kinds(key)
);

CREATE TABLE journal_event_kinds (
  key TEXT PRIMARY KEY CONSTRAINT journal_event_kinds_key_not_empty CHECK (length(btrim(key)) > 0),
  schema_key TEXT REFERENCES schema_documents(key),
  description TEXT NOT NULL CONSTRAINT journal_event_kinds_description_not_empty CHECK (length(btrim(description)) > 0)
);

CREATE TABLE channel_entries (
  id TEXT PRIMARY KEY CONSTRAINT channel_entries_id_not_empty CHECK (length(btrim(id)) > 0),
  channel_key TEXT NOT NULL CONSTRAINT channel_entries_channel_key_not_empty CHECK (length(btrim(channel_key)) > 0),
  work_item_id TEXT NOT NULL CONSTRAINT channel_entries_work_item_id_not_empty CHECK (length(btrim(work_item_id)) > 0),
  enqueued_at TIMESTAMPTZ NOT NULL,
  available_at TIMESTAMPTZ NOT NULL,
  FOREIGN KEY (channel_key) REFERENCES channels(key)
);

CREATE TABLE leases (
  id TEXT PRIMARY KEY CONSTRAINT leases_id_not_empty CHECK (length(btrim(id)) > 0),
  channel_entry_id TEXT NOT NULL UNIQUE CONSTRAINT leases_channel_entry_id_not_empty CHECK (length(btrim(channel_entry_id)) > 0),
  work_item_id TEXT NOT NULL CONSTRAINT leases_work_item_id_not_empty CHECK (length(btrim(work_item_id)) > 0),
  channel_key TEXT NOT NULL CONSTRAINT leases_channel_key_not_empty CHECK (length(btrim(channel_key)) > 0),
  granted_at TIMESTAMPTZ NOT NULL,
  expires_at TIMESTAMPTZ NOT NULL,
  FOREIGN KEY (channel_entry_id) REFERENCES channel_entries(id),
  FOREIGN KEY (channel_key) REFERENCES channels(key)
);

CREATE TABLE journal_events (
  id TEXT PRIMARY KEY CONSTRAINT journal_events_id_not_empty CHECK (length(btrim(id)) > 0),
  work_item_id TEXT NOT NULL CONSTRAINT journal_events_work_item_id_not_empty CHECK (length(btrim(work_item_id)) > 0),
  event_kind_key TEXT NOT NULL CONSTRAINT journal_events_event_kind_key_not_empty CHECK (length(btrim(event_kind_key)) > 0),
  appended_at TIMESTAMPTZ NOT NULL,
  append_index BIGINT GENERATED ALWAYS AS IDENTITY NOT NULL,
  payload JSONB NOT NULL,
  FOREIGN KEY (event_kind_key) REFERENCES journal_event_kinds(key)
);

CREATE INDEX journal_events_work_item_append_order ON journal_events (work_item_id, append_index);
CREATE INDEX channel_entries_ready_order ON channel_entries (channel_key, available_at, enqueued_at, id);
