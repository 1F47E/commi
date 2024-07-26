#!/bin/bash

# Get the current date in the format YYYY-MM-DD
current_date=$(date +%Y-%m-%d)
echo -e "Building \033[0;31mcommi...\033[0m"
go build -ldflags "-X main.version=$(git describe --tags --always --dirty)-${current_date} -X main.errorLoggingOnly=true" -o commi
echo "Installing..."
sudo mv commi /usr/local/bin/
sudo chmod +x /usr/local/bin/commi

