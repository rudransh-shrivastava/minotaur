build:
	go build -o bin/minotaur

run: build
	./bin/minotaur

test:
	go test -v ./... -count=1