#!/bin/bash

# Define colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

echo -e "${BOLD}${BLUE}=== Testing Interactive Mode Validation ===${NC}"
echo -e "${CYAN}This script will help you manually test the validation in interactive mode.${NC}"
echo -e "${CYAN}Follow the prompts and enter invalid values to test validation.${NC}"
echo ""

echo -e "${BOLD}${YELLOW}1. Testing Key Name Validation${NC}"
echo -e "${YELLOW}   - Enter an invalid key name (e.g., 'invalid@name')${NC}"
echo -e "${MAGENTA}   - Validation should fail with an error message${NC}"
echo ""

echo -e "${BOLD}${YELLOW}2. Testing IP Address Validation${NC}"
echo -e "${YELLOW}   - Enter an invalid IP address (e.g., 'not.an.ip')${NC}"
echo -e "${MAGENTA}   - Validation should fail with an error message${NC}"
echo ""

echo -e "${BOLD}${YELLOW}3. Testing GRPC Address Validation${NC}"
echo -e "${YELLOW}   - Enter an invalid GRPC address (e.g., 'localhost' without port)${NC}"
echo -e "${MAGENTA}   - Validation should fail with an error message${NC}"
echo ""

echo -e "${CYAN}Press Enter to start the test...${NC}"
read

# Force cleanup of previous config
echo -e "${BLUE}Running initialization with force flag...${NC}"
./release/supernode-linux-amd64 init --force

echo ""
echo -e "${BOLD}${GREEN}Test completed. Check if validation errors were displayed correctly.${NC}"