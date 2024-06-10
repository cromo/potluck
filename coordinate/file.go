package coordinate

import (
	"log"
	"path/filepath"

	"github.com/cromo/potluck/persistence"
	"github.com/cromo/potluck/transfer"
	"github.com/fsnotify/fsnotify"
)

func File(dir string, db *persistence.HashDB, transferRequests chan<- transfer.Request, done <-chan struct{}) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	watcherDone := make(chan struct{})

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Has(fsnotify.Write) {
					hash := filepath.Base(event.Name)
					if db.HaveHashHexString(hash) {
						transferRequests <- transfer.Request{Hash: hash}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			case <-watcherDone:
				return
			}
		}
	}()

	err = watcher.AddWith(dir)
	if err != nil {
		log.Fatal(err)
	}

	<-done
	watcherDone <- struct{}{}
}
