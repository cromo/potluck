package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
	_ "github.com/glebarez/go-sqlite"
)

const (
	indexUpdated = "INDEX_UPDATED"
)

func main() {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("%s\n\n", workingDir)

	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`create table if not exists content (
		path text primary key not null,
		hash blob not null
	)`)
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})

	indexStatus := make(chan string)
	go index(workingDir, db, indexStatus, done)

	transferRequests := make(chan string)
	go coordinate(db, transferRequests, done)

	<-indexStatus

	var path string
	var hash []byte
	rows, err := db.Query(`select path, hash from content order by hash`)
	if err != nil {
		log.Fatal(err)
	}
	for rows.Next() {
		rows.Scan(&path, &hash)
		log.Printf("%s %s", hex.EncodeToString(hash), path)
	}

	for {
		select {
		case request := <-transferRequests:
			log.Println("Got request for:", request)
		}
	}
	// done <- struct{}{}
	// done <- struct{}{}
}

func index(dir string, db *sql.DB, status chan<- string, done <-chan struct{}) {
	walk(dir, "", db)
	status <- indexUpdated
	<-done
}

func coordinate(db *sql.DB, transferRequests chan<- string, done <-chan struct{}) {
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

func walk(baseDir string, subDir string, db *sql.DB) {
	dir := filepath.Join(baseDir, subDir)
	d, err := os.Open(dir)
	if err != nil {
		log.Fatal("Failed to open directory")
	}
	defer d.Close()

	files, err := d.ReadDir(-1)
	if err != nil {
		log.Fatal("Failed to read directory")
	}
	for _, file := range files {
		fullPath := filepath.Join(dir, file.Name())
		if file.IsDir() {
			walk(baseDir, filepath.Join(subDir, file.Name()), db)
		} else {
			_, err := db.Exec(`insert into content (path, hash) VALUES (?, ?)`, filepath.Join(subDir, file.Name()), hashFile(fullPath))
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func hashFile(path string) []byte {
	content, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	hash := sha256.Sum256(content)
	return hash[:]
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
