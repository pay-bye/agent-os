package kernel

import (
	"context"
	"database/sql"
	"github.com/pay-bye/agent-os/internal/journal"
	"github.com/pay-bye/agent-os/internal/journal/payloads"
	root "github.com/pay-bye/agent-os/internal/kernel"
	"github.com/pay-bye/agent-os/internal/kernel/pause"
	"github.com/pay-bye/agent-os/internal/registry"
)

type pauseFacts struct {
	tx *sql.Tx
}

func (f pauseFacts) Target(ctx context.Context, node registry.NodeKey) (registry.Node, error) {
	return newRegistry(f.tx).FindNode(ctx, node)
}

func (f pauseFacts) Candidates(ctx context.Context) ([]pause.Candidate, error) {
	rows, err := f.tx.QueryContext(ctx, `
SELECT n.key,
       EXISTS (
         SELECT 1
         FROM routing_exclusions e
         WHERE e.node_key = n.key
       )
FROM nodes n
ORDER BY n.key`)
	if err != nil {
		return nil, err
	}

	candidates, err := scanCandidateRows(rows)
	if closeErr := rows.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, err
	}
	return candidatesFromRows(ctx, f.tx, candidates)
}

type candidateRow struct {
	key      registry.NodeKey
	excluded bool
}

func pauseCommand(ctx context.Context, tx *sql.Tx, command pause.Command) (root.PauseResult, error) {
	node, err := pause.Validate(ctx, pauseFacts{tx: tx}, command.Node)
	if err != nil {
		return root.PauseResult{}, err
	}
	if err := insertExclusion(ctx, tx, node.Key()); err != nil {
		return root.PauseResult{}, err
	}
	if err := appendExclusion(ctx, tx, command); err != nil {
		return root.PauseResult{}, err
	}
	return root.PauseResult{Paused: true}, nil
}

func scanCandidateRows(rows *sql.Rows) ([]candidateRow, error) {
	var candidates []candidateRow
	for rows.Next() {
		candidate, err := scanCandidateRow(rows)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, rows.Err()
}

func scanCandidateRow(row rowScanner) (candidateRow, error) {
	var key string
	var excluded bool
	if err := row.Scan(&key, &excluded); err != nil {
		return candidateRow{}, err
	}
	return candidateRow{key: registry.NodeKey(key), excluded: excluded}, nil
}

func candidatesFromRows(
	ctx context.Context,
	tx *sql.Tx,
	rows []candidateRow,
) ([]pause.Candidate, error) {
	candidates := make([]pause.Candidate, 0, len(rows))
	for _, row := range rows {
		candidate, err := candidateFromRow(ctx, tx, row)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}
	return candidates, nil
}

func candidateFromRow(ctx context.Context, tx *sql.Tx, row candidateRow) (pause.Candidate, error) {
	node, err := newRegistry(tx).FindNode(ctx, row.key)
	if err != nil {
		return pause.Candidate{}, err
	}
	return pause.Candidate{Node: node, Excluded: row.excluded}, nil
}

func insertExclusion(ctx context.Context, tx *sql.Tx, node registry.NodeKey) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO routing_exclusions (node_key) VALUES ($1)
ON CONFLICT (node_key) DO NOTHING`, node.String())
	return err
}

func appendExclusion(ctx context.Context, tx *sql.Tx, command pause.Command) error {
	payload, err := payloads.ExclusionSet(command.Node)
	if err != nil {
		return err
	}
	_, err = newJournal(tx).Append(ctx, journal.EventInput{
		ID:         command.Event,
		Coordinate: journal.NodeCoordinate(command.Node),
		Kind:       payloads.ExclusionSetKind,
		AppendedAt: command.PausedAt,
		Payload:    payload,
	})
	return err
}
