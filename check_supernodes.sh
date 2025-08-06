#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if required tools are installed
check_dependencies() {
    local missing_deps=()
    
    if ! command -v grpcurl &> /dev/null; then
        missing_deps+=("grpcurl")
    fi
    
    if ! command -v jq &> /dev/null; then
        missing_deps+=("jq")
    fi
    
    if [ ${#missing_deps[@]} -ne 0 ]; then
        echo -e "${RED}Error: Missing required dependencies: ${missing_deps[*]}${NC}"
        echo "Please install them before running this script."
        exit 1
    fi
}

# Function to get the highest height item from an array
get_highest_height_item() {
    echo "$1" | jq -r 'max_by(.height | tonumber)'
}

# Main function
main() {
    echo -e "${YELLOW}Checking Lumera Supernodes...${NC}\n"
    
    # Check dependencies
    check_dependencies
    
    # Fetch supernode list
    echo "Fetching supernode list..."
    response=$(grpcurl grpc.testnet.lumera.io:443 lumera.supernode.Query.ListSuperNodes 2>/dev/null)
    
    if [ $? -ne 0 ] || [ -z "$response" ]; then
        echo -e "${RED}Error: Failed to fetch supernode list${NC}"
        exit 1
    fi
    
    # Extract supernodes array
    supernodes=$(echo "$response" | jq -r '.supernodes[]' 2>/dev/null)
    
    if [ -z "$supernodes" ]; then
        echo -e "${RED}Error: No supernodes found in response${NC}"
        exit 1
    fi
    
    # Process each supernode
    echo "$response" | jq -c '.supernodes[]' | while IFS= read -r node; do
        # Extract basic info
        supernode_account=$(echo "$node" | jq -r '.supernodeAccount // "N/A"')
        validator_address=$(echo "$node" | jq -r '.validatorAddress // "N/A"')
        
        # Get IP address with highest height
        ip_info=$(echo "$node" | jq -r '.prevIpAddresses' | jq -r 'if . == null or length == 0 then null else max_by(.height | tonumber) end')
        if [ "$ip_info" != "null" ] && [ -n "$ip_info" ]; then
            ip_address=$(echo "$ip_info" | jq -r '.address')
            ip_height=$(echo "$ip_info" | jq -r '.height')
        else
            ip_address="N/A"
            ip_height="N/A"
        fi
        
        # Get state with highest height
        state_info=$(echo "$node" | jq -r '.states' | jq -r 'if . == null or length == 0 then null else max_by(.height | tonumber) end')
        if [ "$state_info" != "null" ] && [ -n "$state_info" ]; then
            current_state=$(echo "$state_info" | jq -r '.state')
            state_height=$(echo "$state_info" | jq -r '.height')
        else
            current_state="N/A"
            state_height="N/A"
        fi
        
        # Get validator moniker
        moniker="N/A"
        if [ "$validator_address" != "N/A" ]; then
            validator_response=$(grpcurl -d "{\"validator_addr\": \"$validator_address\"}" grpc.testnet.lumera.io:443 cosmos.staking.v1beta1.Query.Validator 2>/dev/null)
            if [ $? -eq 0 ] && [ -n "$validator_response" ]; then
                moniker=$(echo "$validator_response" | jq -r '.validator.description.moniker // "N/A"')
            fi
        fi
        
        # Print supernode info
        echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${GREEN}Supernode Account:${NC} $supernode_account"
        echo -e "${GREEN}Validator Address:${NC} $validator_address"
        echo -e "${GREEN}Validator Moniker:${NC} $moniker"
        echo -e "${GREEN}IP Address:${NC} $ip_address (height: $ip_height)"
        echo -e "${GREEN}Current State:${NC} $current_state (height: $state_height)"
        
        # Perform health check if we have valid IP and account
        if [ "$ip_address" != "N/A" ] && [ "$supernode_account" != "N/A" ] && [ -f "./sncli" ]; then
            echo -e "\n${YELLOW}Performing health check...${NC}"
            
            # Ensure IP has port, default to 4444 if not present
            if [[ ! "$ip_address" =~ : ]]; then
                ip_address="${ip_address}:4444"
            fi
            
            # Run health check
            health_result=$(./sncli --grpc_endpoint "$ip_address" --address "$supernode_account" health-check 2>&1)
            health_status=$?
            
            if [ $health_status -eq 0 ] && [[ "$health_result" =~ "SERVING" ]]; then
                echo -e "${GREEN}✅ Health status: SERVING${NC}"
            else
                echo -e "${RED}❌ Health check failed:${NC}"
                echo "$health_result" | head -n 5
            fi
        else
            if [ ! -f "./sncli" ]; then
                echo -e "\n${YELLOW}⚠️  Health check skipped: sncli not found in current directory${NC}"
            elif [ "$ip_address" == "N/A" ] || [ "$supernode_account" == "N/A" ]; then
                echo -e "\n${YELLOW}⚠️  Health check skipped: Missing IP address or supernode account${NC}"
            fi
        fi
    done
    
    echo -e "\n${YELLOW}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Supernode check completed!${NC}\n"
}

# Run main function
main "$@"