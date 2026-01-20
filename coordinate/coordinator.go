package coordinate

import (
	"context"

	"github.com/cromo/potluck/persistence"
	"github.com/cromo/potluck/transfer"
)

type Coordinator interface {
	Coordinate(
		ctx context.Context,
		db *persistence.HashDB,
		incomingTransferRequests chan<- *transfer.Request,
		outgoingTransferRequests <-chan *transfer.Request)
}
