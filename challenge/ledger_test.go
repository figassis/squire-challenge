package main

import "testing"

func setup(t *testing.T) *Ledger {
	t.Helper()
	db, err := connect()
	if err != nil {
		t.Fatalf("could not connect to the database: %v", err)
	}
	t.Cleanup(db.Close)
	return NewLedger(db)
}

func TestNewLedger(t *testing.T) {
	ledger := setup(t)
	if ledger == nil {
		t.Fatal("expected a ledger")
	}
}
