.PHONY: run build dev seed

run:
	go run .

build:
	go build -o bei-exchange .

dev:
	go run . &

docker:
	docker compose up -d postgres

seed:
	curl -s -X POST http://localhost:8080/api/seed | python3 -m json.tool
