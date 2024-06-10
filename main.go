package main

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cromo/potluck/coordinate"
	"github.com/cromo/potluck/index"
	"github.com/cromo/potluck/persistence"
	"github.com/cromo/potluck/transfer"
)

func main() {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)

	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("%s\n\n", workingDir)

	db, err := persistence.CreateInMemory()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-signals
		log.Println("SIGINT received; performing graceful shutdown...")
		cancel()
	}()

	var workers []func()

	indexStatus := make(chan string)
	transferRequests := make(chan transfer.Request)
	workers = append(workers, func() { index.FileWalker(ctx, workingDir, db, indexStatus) })
	workers = append(workers, func() { coordinate.File(ctx, "coordination", db, transferRequests) })
	workers = append(workers, func() { transfer.File(ctx, "transferred", db, transferRequests) })

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

	<-ctx.Done()
}
