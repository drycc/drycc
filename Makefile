GO_ENV=GOROOT=`readlink -f build/_go`

build:
	script/build-drycc

release:
	script/build-drycc --git-version

clean:
	script/clean-drycc

test: test-unit test-integration

test-unit: build
	env $(GO_ENV) PATH=${PWD}/build/bin:${PATH} go test -race -cover ./...

test-unit-root: test-unit
	sudo -E env $(GO_ENV) PATH=${PWD}/build/bin:${PATH} go test -race -cover ./host/volume/...

test-integration: build
	script/run-integration-tests

.PHONY: build release clean test test-unit test-unit-root test-integration
