.PHONY: check test build run

check:
	@test -z "$$(gofmt -l .)"
	go vet ./...
	go test -race ./...
	cd web && npm run build
	PYTHONPATH=workers/src python -c "from aidi_worker import health; assert health()['status'] == 'ok'"
	PYTHONPATH=workers/src python -m unittest discover -s workers/tests -v

test:
	go test -race ./...
	PYTHONPATH=workers/src python -m unittest discover -s workers/tests -v

build:
	mkdir -p bin
	go build -trimpath -o bin/aidi ./cmd/aidi
	cd web && npm run build
	python -m compileall -q workers/src

run:
	go run ./cmd/aidi
