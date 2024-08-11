package main

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	indexStatus := make(chan string)
	transferRequests := make(chan transfer.Request)

	var indexers []index.Indexer
	var coordinators []coordinate.Coordinator
	var transferers []transfer.Transferer

	indexers = append(indexers, &index.FileWalker{Dir: workingDir, Period: 5 * time.Second})
	coordinators = append(coordinators, &coordinate.FileName{Dir: "coordination"})
	transferers = append(transferers, &transfer.FileCopy{Dir: "transferred"})

	for _, indexer := range indexers {
		go indexer.Index(ctx, db, indexStatus)
	}
	for _, coordinator := range coordinators {
		go coordinator.Coordinate(ctx, db, transferRequests)
	}
	for _, transferer := range transferers {
		go transferer.Transfer(ctx, db, transferRequests)
	}

	previousFileCount := 0
	for {
		select {
		case <-ctx.Done():
			return
		case <-indexStatus:
			fileCount, err := db.GetFileCount(ctx)
			if err != nil {
				log.Fatal(err)
			}
			if fileCount != previousFileCount {
				log.Printf("%d files indexed", fileCount)
				hashSets, err := db.ListAll(ctx)
				if err != nil {
					log.Fatal(err)
				}
				for _, hashSet := range hashSets {
					log.Printf("%s %s %s", hex.EncodeToString(hashSet.Hash), hashSet.LastHashTimestamp, hashSet.Path)
				}
			}
			previousFileCount = fileCount
		}
	}
}
