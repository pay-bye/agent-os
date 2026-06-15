CREATE TABLE schema_documents (
  key TEXT PRIMARY KEY CONSTRAINT key_not_empty CHECK (length(btrim(key)) > 0),
  document JSONB NOT NULL
);

CREATE TABLE item_kinds (
  key TEXT PRIMARY KEY CONSTRAINT key_not_empty CHECK (length(btrim(key)) > 0),
  schema_key TEXT NOT NULL CONSTRAINT schema_key_not_empty CHECK (length(btrim(schema_key)) > 0),
  description TEXT NOT NULL CONSTRAINT description_not_empty CHECK (length(btrim(description)) > 0),
  FOREIGN KEY (schema_key) REFERENCES schema_documents(key)
);

CREATE TABLE need_kinds (
  key TEXT PRIMARY KEY CONSTRAINT key_not_empty CHECK (length(btrim(key)) > 0),
  schema_key TEXT REFERENCES schema_documents(key),
  description TEXT NOT NULL CONSTRAINT description_not_empty CHECK (length(btrim(description)) > 0)
);
