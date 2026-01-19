package cmd

import (
	"context"
	"encoding/hex"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cromo/potluck/coordinate"
	"github.com/cromo/potluck/identity"
	"github.com/cromo/potluck/index"
	"github.com/cromo/potluck/persistence"
	"github.com/cromo/potluck/transfer"
	"github.com/spf13/cobra"
)

var sharedRoot string
var localId string
var peerId string

var rootCmd = &cobra.Command{
	Use:   "potluck",
	Short: "Share files with peers who can also share their own files",
	Long: `Potluck allows multiple instances to connect together to offer up their
own files to others to take, just like a food potluck.`,
	Run: runServe,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&sharedRoot, "share", "", "the directory to share")
	rootCmd.PersistentFlags().StringVar(&localId, "local-id", "", "Use the specified ID for the local node")
	rootCmd.PersistentFlags().StringVar(&peerId, "peer-id", "", "The ID of a peer to connect to")
}

// Adapts from the Cobra command domain to the application core domain.
func runServe(cmd *cobra.Command, args []string) {
	serve(&serveArgs{
		shareRoot: sharedRoot,
		localId:   localId,
		peerId:    peerId,
	})
}

type serveArgs struct {
	shareRoot string
	localId   string
	peerId    string
}

func serve(args *serveArgs) {
	log.Printf("Provided share directory: %s\n", args.shareRoot)

	workingDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("%v", err)
	}
	log.Printf("%s\n\n", workingDir)

	db, err := persistence.CreateInMemory()
	if err != nil {
		log.Fatal(err)
	}

	// TODO(cromo): Figure out how to incorporate an ID into transfer process
	id := identity.GenerateID()
	if args.peerId != "" {
		id = args.peerId
	}
	log.Printf("Peer ID: %s", id)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	cancelOnSigint(cancel)

	indexers, coordinators, transferers := configureWorkers(workingDir)
	launchWorkers(ctx, db, indexers, coordinators, transferers)
}

func cancelOnSigint(cancel context.CancelFunc) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT)
	go func() {
		<-signals
		log.Println("SIGINT received; performing graceful shutdown...")
		cancel()
	}()
}

// Configure all the worker goroutines.
//
// Eventually this will pull all this configuration from the database. Until
// then, this set up is mostly hard-coded and very ad-hoc.
func configureWorkers(shareDir string) ([]index.Indexer, []coordinate.Coordinator, []transfer.Transferer) {
	var indexers []index.Indexer
	var coordinators []coordinate.Coordinator
	var transferers []transfer.Transferer

	indexers = append(indexers, &index.FileWalker{Dir: shareDir, Period: 5 * time.Second})
	coordinators = append(coordinators, &coordinate.FileName{Dir: "coordination"})
	transferers = append(transferers, &transfer.FileCopy{Dir: "transferred"})

	return indexers, coordinators, transferers
}

// Runs all workers in separate goroutines with shared context.
func launchWorkers(ctx context.Context, db *persistence.HashDB, indexers []index.Indexer, coordinators []coordinate.Coordinator, transferers []transfer.Transferer) {
	indexStatus := make(chan string)
	transferRequests := make(chan transfer.Request)
	for _, indexer := range indexers {
		go indexer.Index(ctx, db, indexStatus)
	}
	for _, coordinator := range coordinators {
		go coordinator.Coordinate(ctx, db, transferRequests)
	}
	for _, transferer := range transferers {
		go transferer.Transfer(ctx, db, transferRequests)
	}

	// This may become its own worker type at some point in the future, but will
	// likely be removed instead.
	debugMonitor(ctx, db, indexStatus)
}

// Monitors changes to the file index and logs all files when the index changes.
func debugMonitor(ctx context.Context, db *persistence.HashDB, indexStatus <-chan string) {
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
