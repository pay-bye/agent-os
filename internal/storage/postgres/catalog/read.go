package catalog

import (
	"context"
	"database/sql"

	"github.com/pay-bye/agent-os/internal/storage/postgres"
)

type catalogQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func Read(ctx context.Context, querier catalogQuerier) (postgres.Catalog, error) {
	schemas, err := readSchemas(ctx, querier)
	if err != nil {
		return postgres.Catalog{}, err
	}
	items, err := readItems(ctx, querier)
	if err != nil {
		return postgres.Catalog{}, err
	}
	needs, err := readNeeds(ctx, querier)
	if err != nil {
		return postgres.Catalog{}, err
	}
	nodes, err := readNodes(ctx, querier)
	if err != nil {
		return postgres.Catalog{}, err
	}
	routes, err := readRoutes(ctx, querier)
	if err != nil {
		return postgres.Catalog{}, err
	}
	exclusions, err := readRoutingExclusions(ctx, querier)
	if err != nil {
		return postgres.Catalog{}, err
	}
	return postgres.Catalog{
		Schemas:           schemas,
		Items:             items,
		Needs:             needs,
		Nodes:             nodes,
		Routes:            routes,
		RoutingExclusions: exclusions,
	}, nil
}

func readSchemas(ctx context.Context, querier catalogQuerier) ([]postgres.SchemaRecord, error) {
	rows, err := querier.QueryContext(ctx, selectSchemas)
	if err != nil {
		return nil, err
	}
	return postgres.DecodeSchemas(rows)
}

func readItems(ctx context.Context, querier catalogQuerier) ([]postgres.ItemRecord, error) {
	rows, err := querier.QueryContext(ctx, selectItems)
	if err != nil {
		return nil, err
	}
	return postgres.DecodeItems(rows)
}

func readNeeds(ctx context.Context, querier catalogQuerier) ([]postgres.NeedRecord, error) {
	rows, err := querier.QueryContext(ctx, selectNeeds)
	if err != nil {
		return nil, err
	}
	return postgres.DecodeNeeds(rows)
}

func readNodes(ctx context.Context, querier catalogQuerier) ([]postgres.NodeRecord, error) {
	rows, err := querier.QueryContext(ctx, selectNodes)
	if err != nil {
		return nil, err
	}
	records, err := postgres.DecodeNodes(rows)
	if err != nil {
		return nil, err
	}
	for index := range records {
		accepts, err := readAccepts(ctx, querier, records[index].Key)
		if err != nil {
			return nil, err
		}
		records[index].Accepts = accepts
	}
	return records, nil
}

func readAccepts(ctx context.Context, querier catalogQuerier, node string) ([]string, error) {
	rows, err := querier.QueryContext(ctx, selectNodeAccepts, node)
	if err != nil {
		return nil, err
	}
	return postgres.DecodeAccepts(rows)
}

func readRoutes(ctx context.Context, querier catalogQuerier) ([]postgres.RouteRecord, error) {
	rows, err := querier.QueryContext(ctx, selectRoutes)
	if err != nil {
		return nil, err
	}
	return postgres.DecodeRoutes(rows)
}

func readRoutingExclusions(ctx context.Context, querier catalogQuerier) ([]postgres.RoutingExclusionRecord, error) {
	rows, err := querier.QueryContext(ctx, selectRoutingExclusions)
	if err != nil {
		return nil, err
	}
	return postgres.DecodeRoutingExclusions(rows)
}
