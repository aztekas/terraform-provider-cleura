
# Run acceptance tests
.PHONY: testacc
default:
	ls -l
# Run acceptance tests
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m
install-dev:
	GOBIN=~/.terraform.d/plugins/aztek.no/aai/cleura/0.0.1/linux_amd64 go install .
