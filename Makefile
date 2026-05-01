.PHONY: migrate-up migrate-down seed build run

migrate-up:
	@for f in $$(ls migrations/*.up.sql | sort); do \
		echo "Running $$f..."; \
		psql $(DATABASE_URL) -f $$f; \
	done

migrate-down:
	@LAST=$$(ls migrations/*.down.sql | sort -r | head -1); \
	echo "Rolling back $$LAST..."; \
	psql $(DATABASE_URL) -f $$LAST

seed:
	psql $(DATABASE_URL) -f seeds/seed.sql

build:
	go build -o bin/server ./...

run:
	go run .
