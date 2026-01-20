package transfer

import (
	"context"

	"github.com/cromo/potluck/persistence"
)

type Transferer interface {
	Transfer(
		ctx context.Context,
		db *persistence.HashDB,
		transferRequests <-chan *Request)
}
