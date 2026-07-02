package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Ledger struct {
	db *pgxpool.Pool
}

func NewLedger(db *pgxpool.Pool) *Ledger {
	return &Ledger{db: db}
}

func main() {
	db, err := connect()
	if err != nil {
		log.Fatalf("could not connect to the database: %v", err)
	}
	defer db.Close()

	_ = NewLedger(db)

	fmt.Println("application started")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop
	fmt.Println("shutting down")
}
