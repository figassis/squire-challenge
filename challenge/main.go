package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

const dsn string = "postgresql://postgres@localhost:5432/squire?sslmode=disable"

// feeRate is the percentage Squire collects from the gross amount.
const feeRate float64 = 0.05

// squireAccountID is the account that collects fees.
const squireAccountID int = 2 // TODO: set real Squire account id

type paymentPayload struct {
	FromAccountID int     `json:"from_account_id"`
	ToAccountID   int     `json:"to_account_id"`
	Amount        float64 `json:"amount"`
}

type transactionRequest struct {
	FromAccountID   int     `json:"from_account_id"`
	ToAccountID     int     `json:"to_account_id"`
	Amount          float64 `json:"amount"`
	TransactionType string  `json:"transaction_type"` // "payment" | "refund"
}

// ledgerEntry mirrors a row in the transactions table.
type ledgerEntry struct {
	TransactionID string
	AccountID     int64
	EntryID       string
	Amount        float64
	Direction     string // "credit" | "debit"
	Layer         string // "pending" | "settled" | "encumbrance"
}

// TODO: implement double-entry construction from the payload.

func createEntry(accountId int64, amount float64, direction, layer, trxId string) ledgerEntry {
	return ledgerEntry{
		AccountID:     accountId,
		Amount:        amount,
		Direction:     direction,
		Layer:         layer,
		EntryID:       uuid.New().String(),
		TransactionID: trxId,
	}
}

func buildEntries(t transactionRequest) ([]ledgerEntry, error) {

	merchantAmount := t.Amount * (1 - feeRate)
	squireCut := t.Amount - merchantAmount

	trxId := uuid.New().String()
	var ledgerEntries []ledgerEntry
	switch t.TransactionType {
	case "payment":
		ledgerEntries = []ledgerEntry{
			createEntry(3, merchantAmount, "credit", "settled", trxId),
			createEntry(1, squireCut, "debit", "settled", trxId),
			createEntry(2, squireCut, "credit", "settled", trxId),
			createEntry(1, merchantAmount, "debit", "settled", trxId),
			createEntry(1, merchantAmount, "debit", "settled", trxId),
		}
	case "refund":
		ledgerEntries = []ledgerEntry{
			createEntry(3, merchantAmount, "debit", "settled", trxId),
			createEntry(1, squireCut, "credit", "settled", trxId),
			createEntry(2, squireCut, "debit", "settled", trxId),
			createEntry(1, merchantAmount, "credit", "settled", trxId),
		}
	default:
		log.Printf("Unknown transaction type: %s", t.TransactionType)
		return nil, fmt.Errorf("unknown transaction type: %s", t.TransactionType)
	}

	return ledgerEntries, nil

}

// persistEntries writes all entries in a single DB transaction so a payment is
// recorded atomically (all entries commit, or none do).
func persistEntries(db *sql.DB, entries []ledgerEntry) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() // no-op after a successful Commit

	const q = `insert into transactions(entry_id, transaction_id, account_id, amount, direction, layer)
		values($1,$2,$3,$4,$5,$6)`

	for _, e := range entries {
		if _, err := tx.Exec(q, e.EntryID, e.TransactionID, e.AccountID, e.Amount, e.Direction, e.Layer); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func returnTransactions(c fiber.Ctx, db *sql.DB) error {
	rows, err := db.Query("select entry_id, transaction_id, account_id, amount, direction, layer from transactions order by transaction_id")
	if err != nil {
		return fiber.NewError(500, err.Error())
	}
	defer rows.Close()

	entries := []ledgerEntry{}
	for rows.Next() {
		var e ledgerEntry
		if err := rows.Scan(&e.EntryID, &e.TransactionID, &e.AccountID, &e.Amount, &e.Direction, &e.Layer); err != nil {
			return fiber.NewError(500, err.Error())
		}
		entries = append(entries, e)
	}

	return c.JSON(entries)
}

func setupApp() (*fiber.App, *sql.DB, error) {
	app := fiber.New()
	app.Use(recover.New())

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, nil, err
	}

	// Verify the connection is alive
	if err = db.Ping(); err != nil {
		log.Fatal(err)
	}

	return app, db, nil
}

func registerRoutes(app *fiber.App, db *sql.DB) {

	app.Post("/payment", func(c fiber.Ctx) error {

		p := new(paymentPayload)
		if err := c.Bind().All(p); err != nil {
			return fiber.NewError(400, err.Error())
		}

		entries, err := buildEntries(transactionRequest{
			FromAccountID:   p.FromAccountID,
			ToAccountID:     p.ToAccountID,
			Amount:          p.Amount,
			TransactionType: "payment",
		})
		if err != nil {
			return fiber.NewError(500, err.Error())
		}

		// Scaffolding: log instead of hitting the db.
		log.Printf("payment received: from=%d to=%d amount=%.2f -> %d entries",
			p.FromAccountID, p.ToAccountID, p.Amount, len(entries))
		for _, e := range entries {
			log.Printf("  entry: account=%d amount=%.2f direction=%s layer=%s",
				e.AccountID, e.Amount, e.Direction, e.Layer)
		}

		if err := persistEntries(db, entries); err != nil {
			return fiber.NewError(500, err.Error())
		}

		return c.SendString("payment recorded")
	})

	app.Get("/transactions", func(c fiber.Ctx) error {
		return returnTransactions(c, db)
	})
}

func setup2() (*fiber.App, *sql.DB) {
	app, db, err := setupApp()
	if err != nil {
		log.Fatal(err)
	}

	registerRoutes(app, db)

	return app, db
}

func main2() {
	app, db := setup2()
	defer db.Close()
	log.Fatal(app.Listen(":3000"))
}
