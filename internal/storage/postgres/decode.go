package postgres

import (
	"database/sql"
)

type RecordRows interface {
	Next() bool
	Scan(...any) error
	Err() error
	Close() error
}

func DecodeSchemas(rows RecordRows) ([]SchemaRecord, error) {
	defer rows.Close()

	var records []SchemaRecord
	for rows.Next() {
		var record SchemaRecord
		if err := rows.Scan(&record.Key, &record.Document); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func DecodeItems(rows RecordRows) ([]ItemRecord, error) {
	defer rows.Close()

	var records []ItemRecord
	for rows.Next() {
		var record ItemRecord
		if err := rows.Scan(&record.Key, &record.Schema, &record.Description); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func DecodeNeeds(rows RecordRows) ([]NeedRecord, error) {
	defer rows.Close()

	var records []NeedRecord
	for rows.Next() {
		var schema sql.NullString
		var record NeedRecord
		if err := rows.Scan(&record.Key, &schema, &record.Description); err != nil {
			return nil, err
		}
		record.Schema = schema.String
		record.HasSchema = schema.Valid
		records = append(records, record)
	}
	return records, rows.Err()
}

func DecodeNodes(rows RecordRows) ([]NodeRecord, error) {
	defer rows.Close()

	var records []NodeRecord
	for rows.Next() {
		var record NodeRecord
		if err := rows.Scan(&record.Key, &record.Description, &record.Channel, &record.ChannelLabel); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func DecodeAccepts(rows RecordRows) ([]string, error) {
	defer rows.Close()

	var accepts []string
	for rows.Next() {
		var need string
		if err := rows.Scan(&need); err != nil {
			return nil, err
		}
		accepts = append(accepts, need)
	}
	return accepts, rows.Err()
}

func DecodeRoutes(rows RecordRows) ([]RouteRecord, error) {
	defer rows.Close()

	var records []RouteRecord
	for rows.Next() {
		var record RouteRecord
		if err := rows.Scan(&record.Need, &record.Node, &record.Order); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func DecodeRoutingExclusions(rows RecordRows) ([]RoutingExclusionRecord, error) {
	defer rows.Close()

	var records []RoutingExclusionRecord
	for rows.Next() {
		var record RoutingExclusionRecord
		if err := rows.Scan(&record.Node); err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	return records, rows.Err()
}
