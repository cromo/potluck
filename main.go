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

	indexStatus := make(chan string)
	go index.FileWalker(workingDir, db, indexStatus, done)

	transferRequests := make(chan string)
	go coordinate.File(db, transferRequests, done)

	go transfer.File(db, transferRequests, done)

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

	// done <- struct{}{}
	// done <- struct{}{}
	// done <- struct{}{}
	<-make(chan struct{})
}
