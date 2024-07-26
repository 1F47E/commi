
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

