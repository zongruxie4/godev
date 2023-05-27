#!/bin/bash

# Colores
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # Sin color

# Nombre de la aplicación
app_name=$(basename "$(pwd)")

# Generar el ejecutable usando go build
echo -e "${YELLOW}Generando el ejecutable...${NC}"
if go build -o "$HOME/go/bin/$app_name.exe" ./cmd/...; then
  echo -e "${GREEN}El ejecutable ha sido generado.${NC}"
else
  echo -e "${RED}Ocurrió un error al generar el ejecutable.${NC}"
  exit 1
fi

# Verificar si el directorio bin ya está en la variable de entorno PATH
if [[ ":$PATH:" == *":$HOME/go/bin:"* ]]; then
  echo -e "${YELLOW}El directorio bin ya está en la variable de entorno PATH.${NC}"
else
  echo -e "${YELLOW}Agregando el directorio bin a la variable de entorno PATH...${NC}"
  echo "export PATH=\"$HOME/go/bin:$PATH\"" >> ~/.bashrc
  source ~/.bashrc
fi

# Ruta completa del ejecutable
executable_path="$HOME/go/bin/$app_name.exe"

# Verificar si el ejecutable existe antes de reemplazarlo
if [ -f "$executable_path" ]; then
  echo -e "${YELLOW}Reemplazando el ejecutable existente en el directorio bin...${NC}"
  if mv -f "$executable_path" "$HOME/go/bin"; then
    echo -e "${GREEN}El ejecutable ha sido reemplazado en el directorio bin.${NC}"
  else
    echo -e "${RED}Ocurrió un error al reemplazar el ejecutable.${NC}"
    exit 1
  fi
else
  echo -e "${YELLOW}Moviendo el ejecutable al directorio bin...${NC}"
  if mv "$executable_path" "$HOME/go/bin"; then
    echo -e "${GREEN}El ejecutable ha sido movido al directorio bin.${NC}"
  else
    echo -e "${RED}Ocurrió un error al mover el ejecutable.${NC}"
    exit 1
  fi
fi

echo -e "${GREEN}¡Instalación completada!${NC}"
