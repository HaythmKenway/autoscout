#!/bin/bash

# Autoscout Unified Build Script
# Builds both the Go Binary and the Burp Suite Extension (JAR)

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Starting Autoscout Build ===${NC}"

# 1. Build Go Binary
echo -e "${BLUE}[1/2] Building Go Binary...${NC}"
go build -o autoscout main.go
if [ $? -eq 0 ]; then
    echo -e "${GREEN}✔ Go binary 'autoscout' generated successfully.${NC}"
else
    echo "✘ Go build failed."
    exit 1
fi

# 2. Build Burp Extension (Maven)
echo -e "${BLUE}[2/2] Building Burp Extension (JAR)...${NC}"
cd extensions/burp
if command -v mvn &> /dev/null; then
    mvn clean package -DskipTests
    if [ $? -eq 0 ]; then
        JAR_FILE=$(ls target/autoscout-burp-*-jar-with-dependencies.jar | head -n 1)
        echo -e "${GREEN}✔ Burp extension generated: $JAR_FILE${NC}"
    else
        echo "✘ Maven build failed."
        exit 1
    fi
else
    echo "✘ Maven (mvn) not found. Skipping Burp extension build."
    echo "  Please install Maven to build the Java extension."
fi

echo -e "${BLUE}=== Build Complete ===${NC}"
echo -e "Go Binary: ./autoscout"
if [ -f "$JAR_FILE" ]; then
    echo -e "Burp JAR: extensions/burp/$JAR_FILE"
fi
