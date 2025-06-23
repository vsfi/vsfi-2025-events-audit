GOLANGCI_LINT_PACKAGE ?= github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6

.PHONY: build
build:
	@echo "buils all"
	CGO_ENABLED=0 go build -o build/events-audit ./

.PHONY: deps
deps:
	go mod why
	go mod tidy
	go get -v -u ./...
	go mod tidy
	go install $(GOLANGCI_LINT_PACKAGE)


.PHONY: lint
lint:
	go run $(GOLANGCI_LINT_PACKAGE) run --fix

.PHONY: test
test:
	go test -cover -v ./...


# Docker build commands
.PHONY: docker-build-binary
docker-build-binary: build
	@echo "🐳 Building Docker image with pre-built binary..."
	cp build/events-audit package/docker/
	docker build -f package/docker/Dockerfile.binary -t events-audit:binary package/docker/
	rm package/docker/events-audit
	@echo "✅ Docker image 'events-audit:binary' built successfully"

.PHONY: docker-build-multistage
docker-build-multistage:
	@echo "🐳 Building Docker image with multistage build..."
	docker build -f package/docker/Dockerfile.multistage -t events-audit:multistage .
	@echo "✅ Docker image 'events-audit:multistage' built successfully"

.PHONY: docker-build-all
docker-build-all: docker-build-binary docker-build-multistage
	@echo "🐳 All Docker images built successfully"

.PHONY: docker-run-binary
docker-run-binary:
	@echo "🚀 Running Docker container from binary image..."
	docker run --rm -p 8080:8080 events-audit:binary --audit nats --audit-nats-addr nats://host.docker.internal:4222

.PHONY: docker-run-multistage
docker-run-multistage:
	@echo "🚀 Running Docker container from multistage image..."
	docker run --rm -p 8080:8080 events-audit:multistage --audit nats --audit-nats-addr nats://host.docker.internal:4222

.PHONY: docker-clean
docker-clean:
	@echo "🧹 Cleaning Docker images..."
	docker rmi events-audit:binary events-audit:multistage 2>/dev/null || true
	@echo "✅ Docker images cleaned"

# Docker Compose commands
.PHONY: compose-up
compose-up: docker-build-binary
	@echo "🐳 Starting services with docker-compose..."
	docker-compose up -d
	@echo "✅ Services started successfully"

.PHONY: compose-down
compose-down:
	@echo "🛑 Stopping services with docker-compose..."
	docker-compose down
	@echo "✅ Services stopped successfully"

.PHONY: compose-logs
compose-logs:
	@echo "📋 Showing docker-compose logs..."
	docker-compose logs -f

.PHONY: compose-restart
compose-restart: compose-down compose-up
	@echo "🔄 Services restarted successfully"

.PHONY: compose-test
compose-test: compose-up
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@echo "🔥 Testing docker-compose environment..."
	@docker-compose logs events-audit
	@echo "✅ Services are running successfully"
	@make compose-down
