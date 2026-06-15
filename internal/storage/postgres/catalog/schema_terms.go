package catalog

const selectSchemas = `SELECT key, document FROM schema_documents ORDER BY key`

const selectItems = `SELECT key, schema_key, description FROM item_kinds ORDER BY key`

const selectNeeds = `SELECT key, schema_key, description FROM need_kinds ORDER BY key`

const selectNodes = `
SELECT n.key, n.description, c.key, c.description
FROM nodes n
JOIN channels c ON c.node_key = n.key
ORDER BY n.key`

const selectNodeAccepts = `
SELECT need_kind_key
FROM node_capabilities
WHERE node_key = $1
ORDER BY need_kind_key`

const selectRoutes = `
SELECT need_kind_key, node_key, rule_order
FROM routing_rules
ORDER BY need_kind_key, rule_order`

const selectRoutingExclusions = `
SELECT node_key
FROM routing_exclusions
ORDER BY node_key`
