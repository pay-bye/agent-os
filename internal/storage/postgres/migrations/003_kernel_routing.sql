CREATE TABLE work_items (
  id TEXT PRIMARY KEY CONSTRAINT work_items_id_not_empty CHECK (length(btrim(id)) > 0),
  item_kind_key TEXT NOT NULL CONSTRAINT work_items_item_kind_key_not_empty CHECK (length(btrim(item_kind_key)) > 0),
  payload JSONB NOT NULL,
  submitted_at TIMESTAMPTZ NOT NULL,
  FOREIGN KEY (item_kind_key) REFERENCES item_kinds(key)
);

CREATE TABLE routing_rules (
  need_kind_key TEXT NOT NULL CONSTRAINT routing_rules_need_kind_key_not_empty CHECK (length(btrim(need_kind_key)) > 0),
  node_key TEXT NOT NULL CONSTRAINT routing_rules_node_key_not_empty CHECK (length(btrim(node_key)) > 0),
  rule_order INTEGER NOT NULL CONSTRAINT routing_rules_order_positive CHECK (rule_order > 0),
  PRIMARY KEY (need_kind_key, rule_order),
  UNIQUE (need_kind_key, node_key),
  FOREIGN KEY (need_kind_key) REFERENCES need_kinds(key),
  FOREIGN KEY (node_key) REFERENCES nodes(key),
  FOREIGN KEY (node_key, need_kind_key) REFERENCES node_capabilities(node_key, need_kind_key)
);
