#!/bin/bash

# Configuration
REMOTE_HOST="170.205.39.36"
REMOTE_USER="root"
REMOTE_PATH="/root/app/hope_backend"
BINARY_NAME="hope_backend"
LOCAL_BINARY="./${BINARY_NAME}"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}Starting build and deployment process...${NC}"

# Build the Go binary
echo -e "${BLUE}Building Go binary...${NC}"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $BINARY_NAME
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Go build failed${NC}"
    exit 1
fi
echo -e "${GREEN}Build completed successfully${NC}"

# Check if local binary exists and is executable
if [ ! -f "$LOCAL_BINARY" ]; then
    echo -e "${RED}Error: Local binary $LOCAL_BINARY not found${NC}"
    exit 1
fi
chmod +x $LOCAL_BINARY

# Prepare remote environment
ssh "${REMOTE_USER}@${REMOTE_HOST}" << EOF
    # Ensure directory exists with correct permissions
    mkdir -p ${REMOTE_PATH}
    chmod 755 ${REMOTE_PATH}
    
    # Remove existing binary if it exists
    rm -f ${REMOTE_PATH}/${BINARY_NAME}
EOF

if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to prepare remote environment${NC}"
    exit 1
fi

# Copy binary to remote server
echo -e "${BLUE}Copying binary to remote server...${NC}"
scp "$LOCAL_BINARY" "${REMOTE_USER}@${REMOTE_HOST}:${REMOTE_PATH}/"
if [ $? -ne 0 ]; then
    echo -e "${RED}Error: Failed to copy binary to remote server${NC}"
    exit 1
fi

# Execute remote commands
ssh "${REMOTE_USER}@${REMOTE_HOST}" << EOF
    # Set proper permissions
    chmod 755 ${REMOTE_PATH}/${BINARY_NAME}

    # Find and kill existing process
    PID=\$(pgrep ${BINARY_NAME})
    if [ ! -z "\$PID" ]; then
        echo "Stopping existing process (PID: \$PID)..."
        kill \$PID
        sleep 2
        
        # Force kill if process still exists
        if ps -p \$PID > /dev/null; then
            echo "Force stopping process..."
            kill -9 \$PID
        fi
    fi

    # Start new process
    cd ${REMOTE_PATH}
    nohup ./${BINARY_NAME} > ${BINARY_NAME}.log 2>&1 &
    
    # Verify process started
    sleep 2
    NEW_PID=\$(pgrep ${BINARY_NAME})
    if [ ! -z "\$NEW_PID" ]; then
        echo "New process started successfully (PID: \$NEW_PID)"
    else
        echo "Failed to start new process"
        exit 1
    fi
EOF

if [ $? -eq 0 ]; then
    echo -e "${GREEN}Deployment completed successfully${NC}"
else
    echo -e "${RED}Deployment failed${NC}"
    exit 1
fi
