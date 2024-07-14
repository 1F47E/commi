#!/bin/bash

# Get the current date in the format YYYY-MM-DD
current_date=$(date +%Y-%m-%d)

echo "Building aicommit..."
go build -ldflags "-X main.version=$(git describe --tags --always --dirty)-${current_date}" -o aicommit
echo "Installing aicommit..."
sudo mv aicommit /usr/local/bin/
sudo chmod +x /usr/local/bin/aicommit

