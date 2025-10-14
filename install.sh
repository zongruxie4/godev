#!/bin/bash

# Colors
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No color

# Application name
app_name="golite"
# app_name=$(basename "$(pwd)")


# Detect OS and set executable extension
case "$(uname -s)" in
    MINGW*|MSYS*|CYGWIN*|Windows_NT)
        exe_ext=".exe" ;;
    *)
        exe_ext="" ;;
esac


# Generate executable using go build
echo -e "${YELLOW}Generating executable...${NC}"
if go build -o "$HOME/go/bin/$app_name$exe_ext" ./cmd/golite; then
  echo -e "${GREEN}The executable has been generated at $HOME/go/bin/$app_name$exe_ext.${NC}"
else
  echo -e "${RED}An error occurred while generating the executable.${NC}"
  exit 1
fi


# Check if bin directory is already in PATH environment variable
case $SHELL in
    */bash)
        profile_file=~/.bashrc ;;
    */zsh)
        profile_file=~/.zshrc ;;
    *)
        profile_file=~/.profile ;;
esac

if [[ ":$PATH:" == *":$HOME/go/bin:"* ]]; then
  echo -e "${YELLOW}The bin directory is already in the PATH environment variable.${NC}"
else
  echo -e "${YELLOW}Adding bin directory to PATH environment variable...${NC}"
  echo "export PATH=\"$HOME/go/bin:$PATH\"" >> "$profile_file"
  echo -e "${YELLOW}Please restart your terminal or run: source $profile_file${NC}"
fi


# No need to move the executable, it is already in $HOME/go/bin

echo -e "${GREEN}Installation completed!${NC}"