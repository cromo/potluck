package main

import (
	"database/sql"
	"encoding/hex"
	"log"
	"os"

	_ "github.com/glebarez/go-sqlite"

	"github.com/cromo/potluck/coordinate"
	"github.com/cromo/potluck/index"
	"github.com/cromo/potluck/transfer"
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

	var workers []func()

	indexStatus := make(chan string)
	workers = append(workers, func() {
		index.FileWalker(workingDir, db, indexStatus, done)
	})

	transferRequests := make(chan string)
	workers = append(workers, func() { coordinate.File("coordination", db, transferRequests, done) })

	workers = append(workers, func() { transfer.File("transferred", db, transferRequests, done) })

	for _, worker := range workers {
		go worker()
	}

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

	// for range workers {
	// 	done <- struct{}{}
	// }
	<-make(chan struct{})
}
