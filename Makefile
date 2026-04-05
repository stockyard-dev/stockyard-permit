build:
	CGO_ENABLED=0 go build -o permit ./cmd/permit/

run: build
	./permit

test:
	go test ./...

clean:
	rm -f permit

.PHONY: build run test clean
