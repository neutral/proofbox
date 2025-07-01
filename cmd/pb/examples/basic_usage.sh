#!/bin/bash
# Basic usage example for ProofBox CLI

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

DB="example.db"

echo -e "${BLUE}ProofBox CLI Example${NC}"
echo "===================="
echo

# Clean up any existing database
rm -rf $DB

# Initialize database
echo -e "${GREEN}1. Initializing database...${NC}"
pb init --db $DB --json | jq .
echo

# Insert some key-value pairs
echo -e "${GREEN}2. Inserting data...${NC}"
pb put alice 1000 --db $DB --json | jq .
pb put bob 2000 --db $DB --json | jq .
pb put charlie 3000 --db $DB --json | jq .
echo

# Retrieve values
echo -e "${GREEN}3. Retrieving values...${NC}"
echo "Alice's balance:"
pb get alice --db $DB --json | jq .
echo

# Get root hash
echo -e "${GREEN}4. Current root hash...${NC}"
pb root --db $DB --json | jq .
echo

# Generate and verify proof
echo -e "${GREEN}5. Generating proof for Alice...${NC}"
pb prove alice --db $DB --output alice.proof
pb verify alice.proof --json | jq .
echo

# Update a value
echo -e "${GREEN}6. Updating Alice's balance...${NC}"
pb put alice 1500 --db $DB --json | jq .
echo

# Show version history
echo -e "${GREEN}7. Version history...${NC}"
echo "Current version:"
pb root --db $DB --json | jq .version
echo "Alice at version 1:"
pb get alice --db $DB --version 1 --json | jq .value
echo "Alice at latest version:"
pb get alice --db $DB --json | jq .value
echo

# Delete a key
echo -e "${GREEN}8. Deleting Bob...${NC}"
pb delete bob --db $DB --json | jq .
echo

# Verify Bob is deleted
echo -e "${GREEN}9. Verifying Bob is deleted...${NC}"
pb get bob --db $DB --json | jq .
echo

# Generate proof for non-existent key
echo -e "${GREEN}10. Proof for non-existent key...${NC}"
pb prove bob --db $DB --output bob_exclusion.proof
pb verify bob_exclusion.proof --json | jq .
echo

# Show final statistics
echo -e "${GREEN}11. Final statistics...${NC}"
pb stats --db $DB --json | jq .
echo

# Clean up
echo -e "${GREEN}Cleaning up...${NC}"
rm -rf $DB alice.proof bob_exclusion.proof

echo -e "${GREEN}Done!${NC}"