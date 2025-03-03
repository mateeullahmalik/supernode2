#!/bin/bash

# GitHub API URL for the latest release
GITHUB_API_URL="https://api.github.com/repos/LumeraProtocol/lumera/releases/latest"

echo "Fetching latest release information..."

# Make API request to get the latest release information
RELEASE_INFO=$(curl -s -S -L "$GITHUB_API_URL")

# Check if the curl command succeeded
if [ $? -ne 0 ]; then
    echo "Error: Failed to fetch release information from GitHub API"
    exit 1
fi

# Extract the tag name using jq if available, otherwise fallback to grep
if command -v jq >/dev/null 2>&1; then
    TAG_NAME=$(echo "$RELEASE_INFO" | jq -r '.tag_name')
    DOWNLOAD_URL=$(echo "$RELEASE_INFO" | jq -r '.assets[] | select(.name | test("linux_amd64.tar.gz$")) | .browser_download_url')
else
    TAG_NAME=$(echo "$RELEASE_INFO" | grep -o '"tag_name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    DOWNLOAD_URL=$(echo "$RELEASE_INFO" | grep -o '"browser_download_url"[[:space:]]*:[[:space:]]*"[^"]*linux_amd64\.tar\.gz[^"]*"' | sed 's/.*"browser_download_url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
fi

echo "Latest release: $TAG_NAME"

if [ -z "$DOWNLOAD_URL" ]; then
    echo "Error: Could not find Lumera tarball in the latest release"
    exit 1
fi

# Extract the filename from the URL
FILENAME=$(basename "$DOWNLOAD_URL")

echo "Downloading $FILENAME from $DOWNLOAD_URL"

# Create a temporary directory for working with files
TEMP_DIR=$(mktemp -d)
ORIG_DIR=$(pwd)

# Download the file showing progress
curl -L --progress-bar "$DOWNLOAD_URL" -o "$TEMP_DIR/$FILENAME"

# Check if download was successful
if [ $? -ne 0 ]; then
    echo "Error: Download failed"
    rm -rf "$TEMP_DIR"
    exit 1
fi

echo "Download complete!"

# Change to the temp directory
cd "$TEMP_DIR"

# Extract the tarball here
echo "Extracting tarball..."
tar -xzf "$FILENAME"

# Remove the tarball
rm "$FILENAME"

# Check if install.sh exists and run it
if [ -f "install.sh" ]; then
    echo "Running install script..."
    chmod +x "install.sh"
    sudo ./install.sh
    if [ $? -eq 0 ]; then
        echo "Installation complete!"
        LUMERA_PATH=$(pwd)
        echo "Lumera binary installed from: $LUMERA_PATH"
    else
        echo "Error: Installation script failed"
        cd "$ORIG_DIR"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
else
    echo "No install.sh script found"
    # Look for the binary anyway
    if [ -f "lumerad" ]; then
        chmod +x "lumerad"
        echo "Found lumerad binary in $(pwd)"
    fi
fi

# Return to original directory
cd "$ORIG_DIR"

# Clean up - remove everything
echo "Cleaning up..."
rm -rf "$TEMP_DIR"

echo "All done! Lumera has been installed."
