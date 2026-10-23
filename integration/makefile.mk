GOFAIL_BINARY = $(shell pwd)/gofail

.PHONY: run-all-integration-tests
run-all-integration-tests:
	# we enable all failpoints
	$(MAKE) gofail-enable

	# we compile and execute all integration tests
	# add new integration test targets here
	$(MAKE) run-integration-test-sleep 
	$(MAKE) run-integration-test-server

	# we disable all failpoints
	$(MAKE) gofail-disable
	$(MAKE) clean-gofail

.PHONY: clean-all-integration-tests
clean-all-integration-tests: clean-integration-test-sleep gofail-disable

.PHONY: gofail-enable
gofail-enable: build-gofail
	$(GOFAIL_BINARY) enable ./integration/sleep/failpoints
	$(GOFAIL_BINARY) enable ./integration/server/failpoints

.PHONY: gofail-disable
gofail-disable: build-gofail
	$(GOFAIL_BINARY) disable ./integration/sleep/failpoints
	$(GOFAIL_BINARY) disable ./integration/server/failpoints

# run integration test - server
.PHONY: run-integration-test-server
run-integration-test-server:
	cd ./integration/server && go test -v .

# run integration test - sleep
.PHONY: run-integration-test-sleep
run-integration-test-sleep: build-integration-test-sleep execute-integration-test-sleep clean-integration-test-sleep

.PHONY: build-integration-test-sleep
build-integration-test-sleep:
	cd ./integration/sleep && go build -o integration_test_sleep .

.PHONY: execute-integration-test-sleep
execute-integration-test-sleep:
	cd ./integration/sleep && ./integration_test_sleep

.PHONY: clean-integration-test-sleep
clean-integration-test-sleep:
	cd ./integration/sleep && rm integration_test_sleep

# helper: build/remove gofail binaries
.PHONY: build-gofail
build-gofail:
	GO_BUILD_FLAGS="-v" ./build.sh

.PHONY: clean-gofail
clean-gofail:
	rm -f gofail
