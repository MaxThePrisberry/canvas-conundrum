# Canvas Conundrum Server - Test Suite Makefile

# Variables
GOTEST = go test
GOCOVER = go tool cover
BINARY_NAME = canvas-conundrum-server
COVERAGE_FILE = coverage.out
COVERAGE_HTML = coverage.html

# Colors for output
RED = \033[0;31m
GREEN = \033[0;32m
YELLOW = \033[0;33m
NC = \033[0m # No Color

.PHONY: all test unit integration e2e coverage clean build run help

# Default target
all: test

# Help target
help:
	@echo "Canvas Conundrum Test Suite"
	@echo ""
	@echo "Available targets:"
	@echo "  ${GREEN}test${NC}         - Run all tests"
	@echo "  ${GREEN}unit${NC}         - Run unit tests only"
	@echo "  ${GREEN}integration${NC}  - Run integration tests only"
	@echo "  ${GREEN}e2e${NC}          - Run end-to-end tests only"
	@echo "  ${GREEN}coverage${NC}     - Run tests with coverage report"
	@echo "  ${GREEN}coverage-html${NC}- Generate HTML coverage report"
	@echo "  ${GREEN}benchmark${NC}    - Run benchmarks"
	@echo "  ${GREEN}race${NC}         - Run tests with race detector"
	@echo "  ${GREEN}clean${NC}        - Clean test cache and coverage files"
	@echo "  ${GREEN}build${NC}        - Build the server binary"
	@echo "  ${GREEN}run${NC}          - Run the server"
	@echo "  ${GREEN}lint${NC}         - Run linters"
	@echo "  ${GREEN}fmt${NC}          - Format code"

# Run all tests
test:
	@echo "${YELLOW}Running all tests...${NC}"
	@$(GOTEST) -v ./...
	@echo "${GREEN}✓ All tests passed${NC}"

# Run unit tests only
unit:
	@echo "${YELLOW}Running unit tests...${NC}"
	@$(GOTEST) -v ./models/... ./services/... ./utils/... ./config/... ./constants/...
	@echo "${GREEN}✓ Unit tests passed${NC}"

# Run integration tests only
integration:
	@echo "${YELLOW}Running integration tests...${NC}"
	@$(GOTEST) -v ./integration_tests/...
	@echo "${GREEN}✓ Integration tests passed${NC}"

# Run end-to-end tests only
e2e:
	@echo "${YELLOW}Running end-to-end tests...${NC}"
	@$(GOTEST) -v ./e2e_tests/...
	@echo "${GREEN}✓ End-to-end tests passed${NC}"

# Run tests with coverage
coverage:
	@echo "${YELLOW}Running tests with coverage...${NC}"
	@$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@$(GOCOVER) -func=$(COVERAGE_FILE)
	@echo "${GREEN}✓ Coverage report generated${NC}"

# Generate HTML coverage report
coverage-html: coverage
	@echo "${YELLOW}Generating HTML coverage report...${NC}"
	@$(GOCOVER) -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "${GREEN}✓ HTML coverage report generated: $(COVERAGE_HTML)${NC}"

# Run benchmarks
benchmark:
	@echo "${YELLOW}Running benchmarks...${NC}"
	@$(GOTEST) -bench=. -benchmem ./...
	@echo "${GREEN}✓ Benchmarks completed${NC}"

# Run tests with race detector
race:
	@echo "${YELLOW}Running tests with race detector...${NC}"
	@$(GOTEST) -race -v ./...
	@echo "${GREEN}✓ No race conditions detected${NC}"

# Run specific test by name
test-name:
	@if [ -z "$(NAME)" ]; then \
		echo "${RED}Please specify test name: make test-name NAME=TestFunctionName${NC}"; \
		exit 1; \
	fi
	@echo "${YELLOW}Running test: $(NAME)${NC}"
	@$(GOTEST) -v -run $(NAME) ./...

# Run tests for a specific package
test-pkg:
	@if [ -z "$(PKG)" ]; then \
		echo "${RED}Please specify package: make test-pkg PKG=models${NC}"; \
		exit 1; \
	fi
	@echo "${YELLOW}Running tests for package: $(PKG)${NC}"
	@$(GOTEST) -v ./$(PKG)/...

# Clean test cache and coverage files
clean:
	@echo "${YELLOW}Cleaning test cache and coverage files...${NC}"
	@go clean -testcache
	@rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	@rm -f $(BINARY_NAME)
	@echo "${GREEN}✓ Cleaned${NC}"

# Build the server
build:
	@echo "${YELLOW}Building server...${NC}"
	@go build -o $(BINARY_NAME) .
	@echo "${GREEN}✓ Built: $(BINARY_NAME)${NC}"

# Run the server
run: build
	@echo "${YELLOW}Starting server...${NC}"
	@./$(BINARY_NAME)

# Run linters
lint:
	@echo "${YELLOW}Running linters...${NC}"
	@if command -v golangci-lint &> /dev/null; then \
		golangci-lint run; \
	else \
		echo "${YELLOW}golangci-lint not installed, running go vet...${NC}"; \
		go vet ./...; \
	fi
	@echo "${GREEN}✓ Linting completed${NC}"

# Format code
fmt:
	@echo "${YELLOW}Formatting code...${NC}"
	@go fmt ./...
	@echo "${GREEN}✓ Code formatted${NC}"

# Install test dependencies
deps:
	@echo "${YELLOW}Installing test dependencies...${NC}"
	@go get -t ./...
	@go mod tidy
	@echo "${GREEN}✓ Dependencies installed${NC}"

# Quick test (unit tests only, no coverage)
quick:
	@echo "${YELLOW}Running quick tests...${NC}"
	@$(GOTEST) -short ./models/... ./services/... ./utils/...
	@echo "${GREEN}✓ Quick tests passed${NC}"

# Continuous integration target
ci: fmt lint test coverage
	@echo "${GREEN}✓ CI checks passed${NC}"

# Watch for changes and run tests (requires entr)
watch:
	@if command -v entr &> /dev/null; then \
		echo "${YELLOW}Watching for changes...${NC}"; \
		find . -name "*.go" | entr -c make quick; \
	else \
		echo "${RED}entr not installed. Install with: brew install entr (macOS) or apt-get install entr (Linux)${NC}"; \
		exit 1; \
	fi

# Test WebSocket events coverage
test-events:
	@echo "${YELLOW}Testing WebSocket event coverage...${NC}"
	@echo "Checking for all 59 WebSocket events..."
	@go run test_helpers/event_coverage_checker.go
	@echo "${GREEN}✓ Event coverage verified${NC}"

# Performance test
perf:
	@echo "${YELLOW}Running performance tests...${NC}"
	@$(GOTEST) -v -run=XXX -bench=. -benchtime=10s ./...
	@echo "${GREEN}✓ Performance tests completed${NC}"

# Generate test report
report: coverage
	@echo "${YELLOW}Generating test report...${NC}"
	@echo "=== Test Coverage Report ===" > test_report.txt
	@echo "" >> test_report.txt
	@$(GOCOVER) -func=$(COVERAGE_FILE) >> test_report.txt
	@echo "" >> test_report.txt
	@echo "=== Test Results ===" >> test_report.txt
	@$(GOTEST) -v ./... 2>&1 | tee -a test_report.txt
	@echo "${GREEN}✓ Test report generated: test_report.txt${NC}"

# Docker test (runs tests in Docker container)
docker-test:
	@echo "${YELLOW}Running tests in Docker...${NC}"
	@docker build -t canvas-conundrum-test -f Dockerfile.test .
	@docker run --rm canvas-conundrum-test
	@echo "${GREEN}✓ Docker tests completed${NC}"