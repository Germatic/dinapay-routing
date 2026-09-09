package postgres

import (
	"context"
	_ "embed"
	"errors"
	"github.com/Germatic/dinapay-routing/internal/core"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migration.sql
var migration string

type Store struct{ db *pgxpool.Pool }

func New(db *pgxpool.Pool) *Store                  { return &Store{db} }
func (s *Store) Migrate(ctx context.Context) error { _, err := s.db.Exec(ctx, migration); return err }
func (s *Store) Find(ctx context.Context, id string) (string, []byte, bool, error) {
	var hash string
	var body []byte
	err := s.db.QueryRow(ctx, `SELECT request_hash,response FROM dinapay_routing_decisions WHERE request_id=$1`, id).Scan(&hash, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, false, nil
	}
	return hash, body, err == nil, err
}
func (s *Store) Save(ctx context.Context, id, hash string, body []byte) ([]byte, bool, error) {
	tag, err := s.db.Exec(ctx, `INSERT INTO dinapay_routing_decisions(request_id,request_hash,response)VALUES($1,$2,$3)ON CONFLICT DO NOTHING`, id, hash, body)
	if err != nil {
		return nil, false, err
	}
	storedHash, stored, ok, err := s.Find(ctx, id)
	if err != nil {
		return nil, false, err
	}
	if !ok || storedHash != hash {
		return nil, false, core.ErrConflict
	}
	return stored, tag.RowsAffected() == 0, nil
}
func (s *Store) Binding(ctx context.Context, entityType, entityID, provider, connectionID, externalType string) (*core.ProviderBinding, error) {
	var b core.ProviderBinding
	err := s.db.QueryRow(ctx, `SELECT id::text,entity_type,entity_id,external_entity_type,external_entity_id FROM provider_entity_bindings WHERE entity_type=$1 AND entity_id=$2 AND provider_code=$3 AND provider_connection_id=$4 AND external_entity_type=$5 AND status='active'`, entityType, entityID, provider, connectionID, externalType).Scan(&b.BindingID, &b.EntityType, &b.EntityID, &b.ExternalEntityType, &b.ExternalEntityID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return &b, err
}
