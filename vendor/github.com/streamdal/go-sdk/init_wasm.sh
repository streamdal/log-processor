#!/bin/bash

# Step 1: Curl the GitHub API to get the latest release
latest_release=$(curl -s https://api.github.com/repos/streamdal/wasm/releases/latest)

# Step 2: Extract the "browser_download_url" from the JSON response
download_url=$(echo "$latest_release" | grep -o 'https://.*\.zip')

# Step 3: Add debug info
mkdir -p test-assets/wasm
version=$(echo $download_url | cut -d / -f8)
echo "WASM artifact version: ${version}" > test-assets/wasm/version.txt
echo "Last updated: $(date)" >> test-assets/wasm/version.txt

# Step 4: Curl the download URL and save as release.zip
curl -L "$download_url" -o release.zip

# Step 5: Unzip release.zip into the test-assets/wasm/ directory
unzip -o release.zip -d test-assets/wasm/

# Step 6: Clean up & info
rm release.zip
cat test-assets/wasm/version.txt
