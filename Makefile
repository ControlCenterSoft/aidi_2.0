.PHONY: check test build run

check:
	@test -z "$$(gofmt -l .)"
	go vet ./...
	go test -race ./...
	cd web && npm run build
	PYTHONPATH=workers/src python -c "from aidi_worker import health; assert health()['status'] == 'ok'"

test:
	go test -race ./...

build:
	mkdir -p bin
	go build -trimpath -o bin/aidi ./cmd/aidi
	cd web && npm run build
	python -m compileall -q workers/src

run:
	go run ./cmd/aidi
