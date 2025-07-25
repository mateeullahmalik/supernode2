#!/bin/bash

# Test script for init command validation

# Define colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# Build the supernode binary
echo -e "${CYAN}Building supernode binary...${NC}"
make build

# Test cases for keyring backend validation
echo -e "\n\n${BOLD}${BLUE}=== Testing keyring backend validation ===${NC}"
echo -e "${YELLOW}Test 1: Invalid keyring backend${NC}"
./release/supernode-linux-amd64 init -y --keyring-backend=invalid
echo -e "${RED}Expected: Error about invalid keyring backend${NC}"

echo -e "\n${YELLOW}Test 2: Valid keyring backend (test)${NC}"
./release/supernode-linux-amd64 init -y --keyring-backend=test --force
echo -e "${GREEN}Expected: No error about keyring backend${NC}"

# Test cases for key name validation
echo -e "\n\n${BOLD}${BLUE}=== Testing key name validation ===${NC}"
echo -e "${YELLOW}Test 3: Invalid key name with special characters${NC}"
./release/supernode-linux-amd64 init -y --key-name="invalid@name" --force
echo -e "${RED}Expected: Error about invalid key name${NC}"

echo -e "\n${YELLOW}Test 4: Valid key name${NC}"
./release/supernode-linux-amd64 init -y --key-name="valid_name_123" --force
echo -e "${GREEN}Expected: No error about key name${NC}"

# Test cases for IP address validation
echo -e "\n\n${BOLD}${BLUE}=== Testing IP address validation ===${NC}"
echo -e "${YELLOW}Test 5: Invalid IP address${NC}"
./release/supernode-linux-amd64 init -y --supernode-addr="not.an.ip.address" --force
echo -e "${RED}Expected: Error about invalid IP address${NC}"

echo -e "\n${YELLOW}Test 6: Valid IP address${NC}"
./release/supernode-linux-amd64 init -y --supernode-addr="192.168.1.1" --force
echo -e "${GREEN}Expected: No error about IP address${NC}"

# Test cases for GRPC address validation
echo -e "\n\n${BOLD}${BLUE}=== Testing GRPC address validation ===${NC}"
echo -e "${YELLOW}Test 7: Invalid GRPC address (no port)${NC}"
./release/supernode-linux-amd64 init -y --lumera-grpc="localhost" --force
echo -e "${RED}Expected: Error about invalid GRPC address format${NC}"

echo -e "\n${YELLOW}Test 8: Invalid GRPC address (invalid port)${NC}"
./release/supernode-linux-amd64 init -y --lumera-grpc="localhost:invalid" --force
echo -e "${RED}Expected: Error about invalid port in GRPC address${NC}"

echo -e "\n${YELLOW}Test 9: Valid GRPC address (host:port format)${NC}"
./release/supernode-linux-amd64 init -y --lumera-grpc="localhost:9090" --force
echo -e "${GREEN}Expected: No error about GRPC address${NC}"

echo -e "\n${YELLOW}Test 12: Valid GRPC address with schema (schema://host)${NC}"
./release/supernode-linux-amd64 init -y --lumera-grpc="http://example.com" --force
echo -e "${GREEN}Expected: No error about GRPC address${NC}"

echo -e "\n${YELLOW}Test 13: Valid GRPC address with schema and port (schema://host:port)${NC}"
./release/supernode-linux-amd64 init -y --lumera-grpc="https://example.com:8080" --force
echo -e "${GREEN}Expected: No error about GRPC address${NC}"

echo -e "\n${YELLOW}Test 14: Invalid GRPC address with schema (invalid port)${NC}"
./release/supernode-linux-amd64 init -y --lumera-grpc="http://example.com:invalid" --force
echo -e "${RED}Expected: Error about invalid port in GRPC address${NC}"

# Test cases for port validation
echo -e "\n\n${BOLD}${BLUE}=== Testing port validation ===${NC}"
echo -e "${YELLOW}Test 10: Invalid port (too high)${NC}"
./release/supernode-linux-amd64 init -y --supernode-port=70000 --force
echo -e "${RED}Expected: Error about invalid port${NC}"

echo -e "\n${YELLOW}Test 11: Valid port${NC}"
./release/supernode-linux-amd64 init -y --supernode-port=4444 --force
echo -e "${GREEN}Expected: No error about port${NC}"

echo -e "\n\n${BOLD}${GREEN}All tests completed.${NC}"