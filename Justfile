format target:
    gofmt -w $(find . -name '*.go' -not -path './vendor/*')

lint:
    golangci-lint run ./...
docs-build:
    hugo --source docs --cleanDestinationDir

docs-serve:
    hugo server --source docs
