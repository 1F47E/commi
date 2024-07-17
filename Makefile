
verify: lint test	

test:
	@echo "Testing..."
	go test ./... || exit 1
	@echo "Done!"

lint: 
	@echo "Tidy..."
	go mod tidy
	@echo "Linters..."
	golangci-lint run  --color always ./... || exit 1
	@echo "Done!"


run:
	@echo "Running..."
	go run .

install:
	@current_date=$$(date +%Y-%m-%d); \
	echo "Building \033[0;31mcommi...\033[0m"; \
	go build -ldflags "-X main.version=$$(git describe --tags --always --dirty)-$$current_date" -o commi || exit 1; \
	echo "Installing..."; \
	sudo mv commi /usr/local/bin/ && \
	sudo chmod +x /usr/local/bin/commi || { \
		echo "Installation failed. Cleaning up..."; \
		rm -f commi; \
		exit 1; \
	}
