APP_NAME = kontinuous

all: clean deps build

clean:
	@echo "--> Cleaning..."
	@rm -rfv ./build

format:
	@echo "--> Formatting..."
	@go fmt ./...

deps:
	@echo "--> Getting dependencies..."
	@go get -v ./...
	@go get -v github.com/golang/lint/golint

test: format
	@echo "--> Testing..."
	@go test -v ./...

lint:
	@echo "--> Running go lint..."
	golint ./...


build: format
	@echo "--> Building..."
	@mkdir -p build/bin
	@go build -v -o build/bin/${APP_NAME} ./cmd
	@go build -v -o build/bin/${APP_NAME}-cli ./cli

package: build
	@echo "--> Packaging..."
	@mkdir -p build/tar
	@tar czf ./build/tar/${APP_NAME}-`go env GOOS`-`go env GOARCH`.tar.gz ./build/bin/${APP_NAME}
	@tar czf ./build/tar/${APP_NAME}-cli-`go env GOOS`-`go env GOARCH`.tar.gz ./build/bin/${APP_NAME}-cli
