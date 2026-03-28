format target="all":
    gofmt -w $(find . -name '*.go' -not -path './vendor/*')

lint target="all":
    golangci-lint run ./...
docs-build:
    hugo --source docs --cleanDestinationDir

docs-serve:
    hugo server --source docs
