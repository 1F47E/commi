#!/bin/bash

echo -e "\033[1;34m===== COMMI Installation =====\033[0m"
echo -e "This installation script will:"
echo -e "  1. Build commi from source with version information"
echo -e "  2. Install the binary to \033[0;33m/usr/local/bin/commi\033[0m (requires sudo permission)"
echo -e "  3. Make the binary executable"
echo ""
echo -e "You will be prompted for your password to complete the installation."
echo ""

# Get user confirmation
read -p "Continue with installation? (y/n): " confirm
if [[ "$confirm" != "y" && "$confirm" != "Y" ]]; then
    echo "Installation cancelled."
    exit 0
fi

# Get the current date in the format YYYY-MM-DD
current_date=$(date +%Y-%m-%d)
echo -e "\nBuilding \033[0;31mcommi...\033[0m"
go build -ldflags "-X main.version=$(git describe --tags --always --dirty)-${current_date} -X main.errorLoggingOnly=true" -o commi

echo -e "\nInstalling to /usr/local/bin (requires sudo permission)..."
sudo mv commi /usr/local/bin/
sudo chmod +x /usr/local/bin/commi

echo -e "\n\033[1;32mInstallation completed successfully!\033[0m"
echo -e "You can now run 'commi' from anywhere.\n\nTry commi -v"

