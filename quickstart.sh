#!/bin/bash
# GreenLane Quick Start Script
# Automates the initial setup and starts all services

set -e

BLUE='\033[0;34m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}"
cat << "EOF"
╔═══════════════════════════════════════════════════════════════╗
║                    GreenLane QuickStart                       ║
║          Automated Setup & Launch Script                      ║
╚═══════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Check prerequisites
echo -e "${YELLOW}🔍 Checking prerequisites...${NC}"

if ! command_exists docker; then
    echo -e "${RED}❌ Docker not found. Please install Docker first.${NC}"
    exit 1
fi

if ! command_exists go; then
    echo -e "${RED}❌ Go not found. Please install Go 1.23+ first.${NC}"
    exit 1
fi

if ! command_exists protoc; then
    echo -e "${RED}❌ protoc not found. Please install Protocol Buffers compiler.${NC}"
    exit 1
fi

if ! command_exists cargo; then
    echo -e "${RED}❌ Rust/Cargo not found. Please install Rust.${NC}"
    exit 1
fi

if ! command_exists python3; then
    echo -e "${RED}❌ Python3 not found. Please install Python 3.10+.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ All prerequisites found!${NC}\n"

# Step 1: Start infrastructure
echo -e "${BLUE}📦 Step 1: Starting infrastructure (Redis, Redpanda, TimescaleDB)...${NC}"
make up
echo ""

# Wait for services to be healthy
echo -e "${YELLOW}⏳ Waiting for services to be healthy (30s)...${NC}"
sleep 30
echo -e "${GREEN}✅ Infrastructure started!${NC}\n"

# Step 2: Install Go protobuf tools
echo -e "${BLUE}📝 Step 2: Installing Go protobuf generators...${NC}"
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
echo -e "${GREEN}✅ Protobuf generators installed!${NC}\n"

# Step 3: Generate protobuf code
echo -e "${BLUE}🔨 Step 3: Generating protobuf code...${NC}"
make proto
echo ""

# Step 4: Generate Python protobuf code
echo -e "${BLUE}🐍 Step 4: Generating Python protobuf code...${NC}"
cd simulator
./generate_proto.sh
cd ..
echo -e "${GREEN}✅ Python protobuf generated!${NC}\n"

# Step 5: Install Python dependencies
echo -e "${BLUE}📦 Step 5: Installing Python dependencies...${NC}"
cd simulator
pip3 install -r requirements.txt -q
cd ..
echo -e "${GREEN}✅ Python dependencies installed!${NC}\n"

# Step 6: Download Go dependencies
echo -e "${BLUE}📦 Step 6: Downloading Go dependencies...${NC}"
cd services/ingestion && go mod download && cd ../..
cd services/mock-grid && go mod download && cd ../..
cd cli && go mod download && cd ../..
echo -e "${GREEN}✅ Go dependencies downloaded!${NC}\n"

# Step 7: Build services
echo -e "${BLUE}🔨 Step 7: Building all services...${NC}"
make build-all
echo -e "${GREEN}✅ All services built!${NC}\n"

# Final instructions
echo -e "${GREEN}"
cat << "EOF"
╔═══════════════════════════════════════════════════════════════╗
║                   🎉 Setup Complete! 🎉                       ║
╚═══════════════════════════════════════════════════════════════╝
EOF
echo -e "${NC}"

echo -e "${BLUE}📊 Access Points:${NC}"
echo -e "  • Redpanda Console: ${YELLOW}http://localhost:8080${NC}"
echo -e "  • TimescaleDB: ${YELLOW}postgresql://greenlane:greenlane_password@localhost:5432/greenlane${NC}"
echo -e "  • Redis: ${YELLOW}localhost:6379${NC}"
echo ""

echo -e "${BLUE}🚀 To start the steel thread, open 6 terminals and run:${NC}"
echo ""
echo -e "${YELLOW}Terminal 1:${NC} make dev-mock-grid"
echo -e "${YELLOW}Terminal 2:${NC} make dev-ingestion"
echo -e "${YELLOW}Terminal 3:${NC} cd services/pricing-worker && RUST_LOG=info cargo run --release"
echo -e "${YELLOW}Terminal 4:${NC} make dev-cli"
echo -e "${YELLOW}Terminal 5:${NC} make dev-simulator"
echo -e "${YELLOW}Terminal 6:${NC} make logs"
echo ""

echo -e "${GREEN}📚 For detailed testing instructions, see TESTING.md${NC}"
echo ""
