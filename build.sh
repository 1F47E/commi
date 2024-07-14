echo "Building aicommit..."
go build -o aicommit
echo "Installing aicommit..."
sudo mv aicommit /usr/local/bin/
sudo chmod +x /usr/local/bin/aicommit

