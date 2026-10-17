#!/bin/bash
set -e

echo "Compilando o binário Go..."
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o corsario cmd/corsario/main.go

echo "Gerando pacote .deb com nFPM..."
# Pressupõe que o nfpm está instalado no sistema ou no PATH
nfpm pkg --packager deb --target .
echo "Pronto! corsario_1.0.0_amd64.deb gerado."
