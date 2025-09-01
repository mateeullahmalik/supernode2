#!/bin/bash

# Dynamic Supernode Version Checker Script
# Fetches live supernode data from Lumera testnet API
# Usage: ./check_versions.sh <latest_version>
# Example: ./check_versions.sh v2.2.6

if [ $# -eq 0 ]; then
    echo "Usage: $0 <latest_version>"
    echo "Example: $0 v2.2.6"
    exit 1
fi

LATEST_VERSION="$1"
TOTAL_CHECKED=0
LATEST_VERSION_COUNT=0
UNREACHABLE_COUNT=0
TIMEOUT=2
API_URL="https://lcd.testnet.lumera.io/LumeraProtocol/lumera/supernode/list_super_nodes?pagination.limit=1000&pagination.count_total=true"

# Arrays to track failed calls
FAILED_CALLS=()
INVALID_RESPONSES=()

echo "Fetching supernode list from Lumera API..."
echo "=============================================================================="

# Fetch supernode data from API
echo "Fetching from: $API_URL"
SUPERNODE_DATA=$(curl -s --connect-timeout 10 --max-time 30 \
    -H "User-Agent: Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36" \
    -H "Accept: application/json, text/plain, */*" \
    -H "Accept-Language: en-US,en;q=0.9" \
    -H "Accept-Encoding: gzip, deflate, br" \
    -H "Connection: keep-alive" \
    -H "Upgrade-Insecure-Requests: 1" \
    "$API_URL")
CURL_STATUS=$?

if [ $CURL_STATUS -ne 0 ]; then
    echo "ERROR: curl failed with exit code $CURL_STATUS"
    exit 1
fi

if [ -z "$SUPERNODE_DATA" ]; then
    echo "ERROR: Empty response from API"
    exit 1
fi

# Debug: Show first 200 characters of response
echo "API Response preview: $(echo "$SUPERNODE_DATA" | head -c 200)..."

# Check if response is valid JSON
if ! echo "$SUPERNODE_DATA" | jq empty 2>/dev/null; then
    echo "ERROR: Invalid JSON response from API"
    echo "Full response:"
    echo "$SUPERNODE_DATA"
    exit 1
fi

# Check if jq is available for JSON parsing
if ! command -v jq &> /dev/null; then
    echo "ERROR: jq is required for JSON parsing but not installed"
    echo "Please install jq: sudo apt-get install jq (Ubuntu/Debian) or brew install jq (macOS)"
    exit 1
fi

# Parse JSON and extract supernodes with their latest IP addresses
echo "Parsing supernode data and extracting latest IP addresses..."

# Create temporary file to store processed supernode data
TEMP_FILE=$(mktemp)

# Extract supernode data and find latest IP for each
echo "$SUPERNODE_DATA" | jq -r '.supernodes[] | 
    # Find the address with highest height for each supernode
    (.prev_ip_addresses | sort_by(.height | tonumber) | reverse | .[0].address) as $latest_ip |
    .supernode_account + "," + $latest_ip' > "$TEMP_FILE"

SUPERNODE_COUNT=$(wc -l < "$TEMP_FILE")
echo "Found $SUPERNODE_COUNT supernodes to check"
echo "Starting version checks (target version: $LATEST_VERSION)"
echo "=============================================================================="

# Process each supernode
while IFS=',' read -r ACCOUNT LATEST_IP; do
    TOTAL_CHECKED=$((TOTAL_CHECKED + 1))
    
    # Skip empty lines
    [ -z "$ACCOUNT" ] && continue
    
    # Clean up the IP first - remove all whitespace and trailing spaces
    CLEAN_IP=$(echo "$LATEST_IP" | sed -E 's/[[:space:]]+/ /g' | sed -E 's/^[[:space:]]+|[[:space:]]+$//')
    
    # Convert port from 4444 to 8002 for status API, handle various formats
    STATUS_ENDPOINT=$(echo "$CLEAN_IP" | sed -E '
        s/:4444$/:8002/
        s/:443$/:8002/
        /^https?:\/\//!s/^/http:\/\//
        /:[0-9]+$/!s/$/:8002/
    ')
    
    # For URLs that already have http/https, extract just the domain:port part
    if [[ "$STATUS_ENDPOINT" =~ ^https?:// ]]; then
        # Extract domain from URL and add :8002
        DOMAIN=$(echo "$STATUS_ENDPOINT" | sed -E 's|^https?://([^/]+).*|\1|')
        STATUS_ENDPOINT="http://$DOMAIN"
        # Add port if not present
        if [[ ! "$STATUS_ENDPOINT" =~ :[0-9]+$ ]]; then
            STATUS_ENDPOINT="$STATUS_ENDPOINT:8002"
        fi
    fi
    
    # Make HTTP request to status API
    RESPONSE=$(curl -s --connect-timeout $TIMEOUT --max-time $TIMEOUT "$STATUS_ENDPOINT/api/v1/status" 2>/dev/null)
    CURL_EXIT_CODE=$?
    
    if [ $CURL_EXIT_CODE -eq 0 ] && [ ! -z "$RESPONSE" ]; then
        # Extract version from response
        VERSION=$(echo "$RESPONSE" | jq -r '.version' 2>/dev/null)
        
        if [ ! -z "$VERSION" ] && [ "$VERSION" != "null" ]; then
            if [ "$VERSION" = "$LATEST_VERSION" ]; then
                echo "✓ $ACCOUNT ($STATUS_ENDPOINT): $VERSION (LATEST)"
                LATEST_VERSION_COUNT=$((LATEST_VERSION_COUNT + 1))
            else
                echo "○ $ACCOUNT ($STATUS_ENDPOINT): $VERSION"
            fi
        else
            echo "✗ $ACCOUNT ($STATUS_ENDPOINT): ERROR (invalid response format)"
            INVALID_RESPONSES+=("$ACCOUNT ($STATUS_ENDPOINT)")
            UNREACHABLE_COUNT=$((UNREACHABLE_COUNT + 1))
        fi
    else
        # Determine specific error type
        if [ $CURL_EXIT_CODE -eq 6 ]; then
            ERROR_MSG="DNS resolution failed"
        elif [ $CURL_EXIT_CODE -eq 7 ]; then
            ERROR_MSG="connection refused"
        elif [ $CURL_EXIT_CODE -eq 28 ]; then
            ERROR_MSG="timeout"
        else
            ERROR_MSG="curl error code $CURL_EXIT_CODE"
        fi
        
        echo "✗ $ACCOUNT ($STATUS_ENDPOINT): UNREACHABLE ($ERROR_MSG)"
        FAILED_CALLS+=("$ACCOUNT ($STATUS_ENDPOINT) - $ERROR_MSG")
        UNREACHABLE_COUNT=$((UNREACHABLE_COUNT + 1))
    fi
    
done < "$TEMP_FILE"

# Cleanup temp files
rm "$TEMP_FILE"
rm -rf "$TEMP_DIR"

echo "=============================================================================="
echo "SUMMARY:"
echo "Total supernodes checked: $TOTAL_CHECKED"
echo "Latest version ($LATEST_VERSION): $LATEST_VERSION_COUNT"
echo "Other versions: $((TOTAL_CHECKED - LATEST_VERSION_COUNT - UNREACHABLE_COUNT))"
echo "Unreachable: $UNREACHABLE_COUNT"

if [ $((TOTAL_CHECKED - UNREACHABLE_COUNT)) -gt 0 ]; then
    echo "Percentage with latest version: $(( LATEST_VERSION_COUNT * 100 / (TOTAL_CHECKED - UNREACHABLE_COUNT) ))%"
else
    echo "Percentage: N/A (all unreachable)"
fi

# List all failed calls
if [ ${#FAILED_CALLS[@]} -gt 0 ] || [ ${#INVALID_RESPONSES[@]} -gt 0 ]; then
    echo ""
    echo "FAILED CALLS DETAILS:"
    echo "====================="
    
    if [ ${#FAILED_CALLS[@]} -gt 0 ]; then
        echo "Connection failures (${#FAILED_CALLS[@]}):"
        for failed in "${FAILED_CALLS[@]}"; do
            echo "  - $failed"
        done
    fi
    
    if [ ${#INVALID_RESPONSES[@]} -gt 0 ]; then
        echo "Invalid response format (${#INVALID_RESPONSES[@]}):"
        for invalid in "${INVALID_RESPONSES[@]}"; do
            echo "  - $invalid"
        done
    fi
fi