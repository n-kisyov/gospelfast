.PHONY: build run run-cli test migrate-up migrate-down migrate-create clean dev css

APP=gospelfast
CLI=gospelfast-cli
DB_URL=postgres://gospelfast:gospelfast@localhost:5432/gospelfast?sslmode=disable

build: swagger
	go build -o bin/$(APP) ./cmd/$(APP)/
	go build -o bin/$(CLI) ./cmd/$(CLI)/

swagger:
	swag init -g cmd/$(APP)/main.go -o docs/

run: swagger
	go run ./cmd/$(APP)/

run-cli:
	go run ./cmd/$(CLI)/

test:
	go test ./...

css:
	./tailwindcss -i ./web/static/css/input.css -o ./web/static/css/tailwind.css --minify

css-watch:
	./tailwindcss -i ./web/static/css/input.css -o ./web/static/css/tailwind.css --watch

dev: css
	go run ./cmd/$(APP)/

migrate-up:
	migrate -path internal/db/migrations -database "$(DB_URL)" up

migrate-down:
	migrate -path internal/db/migrations -database "$(DB_URL)" down

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir internal/db/migrations -seq $$name

clean:
	rm -rf bin/
