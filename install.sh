#!/bin/bash

# Colors
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No color

# Application name
app_name="godev"
# app_name=$(basename "$(pwd)")

# Detect OS and set executable extension
if [[ "$OSTYPE" == "msys" || "$OSTYPE" == "win32" ]]; then
    exe_ext=".exe"
else
    exe_ext=""
fi

# Generate executable using go build
echo -e "${YELLOW}Generating executable...${NC}"
if go build -o "$HOME/go/bin/$app_name$exe_ext" ./cmd/...; then
  echo -e "${GREEN}The executable has been generated.${NC}"
else
  echo -e "${RED}An error occurred while generating the executable.${NC}"
  exit 1
fi

# Check if bin directory is already in PATH environment variable
if [[ ":$PATH:" == *":$HOME/go/bin:"* ]]; then
  echo -e "${YELLOW}The bin directory is already in the PATH environment variable.${NC}"
else
  echo -e "${YELLOW}Adding bin directory to PATH environment variable...${NC}"
  echo "export PATH=\"$HOME/go/bin:$PATH\"" >> ~/.bashrc
  source ~/.bashrc
fi

# Full path of the executable
executable_path="$HOME/go/bin/$app_name$exe_ext"

# Check if executable exists before replacing it
if [ -f "$executable_path" ]; then
  echo -e "${YELLOW}Replacing existing executable in bin directory...${NC}"
  if mv -f "$executable_path" "$HOME/go/bin"; then
    echo -e "${GREEN}The executable has been replaced in bin directory.${NC}"
  else
    echo -e "${RED}An error occurred while replacing the executable.${NC}"
    exit 1
  fi
else
  echo -e "${YELLOW}Moving executable to bin directory...${NC}"
  if mv "$executable_path" "$HOME/go/bin"; then
    echo -e "${GREEN}The executable has been moved to bin directory.${NC}"
  else
    echo -e "${RED}An error occurred while moving the executable.${NC}"
    exit 1
  fi
fi

echo -e "${GREEN}Installation completed!${NC}"