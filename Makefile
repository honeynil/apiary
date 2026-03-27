BINARY  := apiary
CMD     := ./cmd/apiary
OUTFILE := openapi.yaml

.PHONY: build test install clean generate lint

build:
	go build -o bin/$(BINARY) $(CMD)

test:
	go test ./...

install:
	go install $(CMD)

clean:
	rm -rf bin/ $(OUTFILE)

# Generate specs for all bundled examples.
generate:
	go run $(CMD) -out docs/sample.yaml  -title "Sample API"              -version "0.1.0" ./testdata/sample
	go run $(CMD) -out docs/tasks.yaml   -title "Task Manager API"        -version "1.0.0" -security bearer ./testdata/router
	go run $(CMD) -out docs/tasks_gin.yaml -title "Task Manager API (gin)" -version "1.0.0" -security bearer ./testdata/gin

lint:
	golangci-lint run ./...
