package index

import (
	"context"

	"github.com/cromo/potluck/persistence"
)

type Indexer interface {
	Index(ctx context.Context, db *persistence.HashDB, status chan<- string)
}
