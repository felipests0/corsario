#!/usr/bin/env bash
set -euo pipefail

BINARY_URL="https://github.com/felipests0/corsario/releases/latest/download/corsario_linux_amd64"
INSTALL_PATH="/usr/local/bin/corsario"
DATA_DIR="/var/lib/corsario"
SERVICE_FILE="/etc/systemd/system/corsario.service"

if [[ "${EUID}" -ne 0 ]]; then
    echo "⚠️  Erro: Este script precisa ser executado como root (use sudo)." >&2
    exit 1
fi

echo "⚓ Iniciando o embarque do Corsário..."

TMP_BIN="$(mktemp)"
echo "📥 Baixando o binário do Corsário de ${BINARY_URL}..."
if command -v curl >/dev/null 2>&1; then
    curl -fsSL "${BINARY_URL}" -o "${TMP_BIN}"
elif command -v wget >/dev/null 2>&1; then
    wget -qO "${TMP_BIN}" "${BINARY_URL}"
else
    echo "⚠️  Erro: curl ou wget é necessário para efetuar o download." >&2
    rm -f "${TMP_BIN}"
    exit 1
fi

mv "${TMP_BIN}" "${INSTALL_PATH}"
chmod +x "${INSTALL_PATH}"

echo "📁 Preparando porão de dados em ${DATA_DIR}..."
mkdir -p "${DATA_DIR}"
chmod 700 "${DATA_DIR}"

echo "⚙️  Configurando o serviço systemd em ${SERVICE_FILE}..."
cat <<EOF > "${SERVICE_FILE}"
[Unit]
Description=Corsário - Painel de Gestão de Containers
After=network.target docker.service
Requires=docker.service

[Service]
Type=simple
WorkingDirectory=${DATA_DIR}
ExecStart=${INSTALL_PATH}
Restart=always
RestartSec=5

# Segurança e Isolamento
ProtectSystem=strict
ProtectHome=yes
PrivateTmp=yes
ReadWritePaths=${DATA_DIR}

[Install]
WantedBy=multi-user.target
EOF

echo "🚀 Ativando e iniciando o serviço corsario.service..."
systemctl daemon-reload
systemctl enable corsario.service
systemctl start corsario.service

SERVER_IP="$(hostname -I 2>/dev/null | awk '{print $1}')"
if [[ -z "${SERVER_IP}" ]]; then
    SERVER_IP="<ip>"
fi

echo ""
echo "================================================================"
echo "⚓ O navio Corsário ancorou com sucesso!"
echo "🌐 Acesse a Ponte de Comando em: https://${SERVER_IP}:9443"
echo "================================================================"
