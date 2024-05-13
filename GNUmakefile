
# Run acceptance tests
.PHONY: docs
default:
	ls -l
# Run acceptance tests
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m
install:
	go install .
snapshot:
	goreleaser build --clean --snapshot
docs:
	go generate
