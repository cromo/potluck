package coordinate

import (
	"database/sql"
	"encoding/hex"
	"log"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

func File(db *sql.DB, transferRequests chan<- string, done <-chan struct{}) {
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
					if have(db, hash) {
						transferRequests <- hash
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

	err = watcher.AddWith("coordination")
	if err != nil {
		log.Fatal(err)
	}

	<-done
	watcherDone <- struct{}{}
}

func have(db *sql.DB, hash string) bool {
	hashBin, err := hex.DecodeString(hash)
	if err != nil {
		log.Printf("Error decoding hash: %v\n", err)
		return false
	}
	result := db.QueryRow(`select path from content where hash = ?`, hashBin)
	var path string
	err = result.Scan(&path)
	return err != sql.ErrNoRows
}
