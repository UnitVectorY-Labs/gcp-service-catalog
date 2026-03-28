
# Commands for gcp-service-catalog
default:
  @just --list
# Build gcp-service-catalog with Go
build:
  go build ./...

# Run tests for gcp-service-catalog with Go
test:
  go clean -testcache
  go test ./...