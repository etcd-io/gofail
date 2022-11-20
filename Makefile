SRCS=$(filter-out %_test.go, $(wildcard *.go */*.go))

.PHONY: all
all: gofail

.PHONY: clean
clean:
	rm -f gofail

.PHONY: test
test:
	go test -v --race -cpu=1,2,4 ./code/ ./runtime/

.PHONY: fmt
fmt:
	gofmt -w -l -s $(SRCS)

gofail: 
	GO_BUILD_FLAGS="-v" ./build.sh
	./gofail --version
