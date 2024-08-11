package coordinate

import (
	"context"
	"log"
	"path/filepath"

	"github.com/cromo/potluck/persistence"
	"github.com/cromo/potluck/transfer"
	"github.com/fsnotify/fsnotify"
)

type FileName struct {
	Dir string
}

func (coordinator *FileName) Coordinate(ctx context.Context, db *persistence.HashDB, transferRequests chan<- transfer.Request) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Has(fsnotify.Write) {
					hash := filepath.Base(event.Name)
					if db.HaveHashHexString(ctx, hash) {
						transferRequests <- transfer.Request{Hash: hash}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()

	err = watcher.AddWith(coordinator.Dir)
	if err != nil {
		log.Fatal(err)
	}

	<-ctx.Done()

}
