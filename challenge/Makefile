.PHONY: start test stop

export DATABASE_URL ?= postgres://ledger:ledger@127.0.0.1:5432/ledger?sslmode=disable

start:
	docker compose up -d --wait
	go build -o bin/ledger .
	@./bin/ledger & echo $$! > .app.pid
	@sleep 1
	@echo "App started (pid $$(cat .app.pid)). Run 'make stop' to shut down."

test:
	docker compose up -d --wait
	go test -race ./...

stop:
	-@if [ -f .app.pid ]; then kill $$(cat .app.pid) 2>/dev/null || true; rm -f .app.pid; fi
	docker compose down
