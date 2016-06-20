SRCS=$(filter-out %_test.go, $(wildcard *.go */*.go))

.PHONY: all
all: gofail

.PHONY: clean
clean:
	rm -f gofail

.PHONY: test
test:
	go test -v --race -cpu=1,2,4 ./code/ ./runtime/

gofail: $(SRCS)
	go build -v