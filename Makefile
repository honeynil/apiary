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

# Generate the spec for the bundled sample handlers.
generate:
	go run $(CMD) -out docs/sample.yaml -title "Sample API" -version "0.1.0" ./testdata/sample

lint:
	golangci-lint run ./...
