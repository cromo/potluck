package main

import (
	"encoding/hex"
	"log"
	"os"

	"github.com/cromo/potluck/coordinate"
	"github.com/cromo/potluck/index"
	"github.com/cromo/potluck/persistence"
	"github.com/cromo/potluck/transfer"
)

func main() {
	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("%s\n\n", workingDir)

	db, err := persistence.CreateInMemory()
	if err != nil {
		log.Fatal(err)
	}

	done := make(chan struct{})

	var workers []func()

	indexStatus := make(chan string)
	transferRequests := make(chan transfer.Request)
	workers = append(workers, func() { index.FileWalker(workingDir, db, indexStatus, done) })
	workers = append(workers, func() { coordinate.File("coordination", db, transferRequests, done) })
	workers = append(workers, func() { transfer.File("transferred", db, transferRequests, done) })

	for _, worker := range workers {
		go worker()
	}

	<-indexStatus

	hashSets, err := db.ListAll()
	if err != nil {
		log.Fatal(err)
	}
	for _, hashSet := range hashSets {
		log.Printf("%s %s", hex.EncodeToString(hashSet.Hash), hashSet.Path)
	}

	// for range workers {
	// 	done <- struct{}{}
	// }
	<-make(chan struct{})
}
