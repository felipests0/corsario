# Projeto Corsário - Palácio da Memória

## 1. Identidade e Temática
- **Nome:** Corsário
- **Tema:** Marítimo, navios, oceanos, frotas. O painel gerencia uma "frota" de containers.
- **Símbolo Visual:** Um navio corsário (século XVII/XVIII) visto de lado. No centro do casco de madeira clássico, há um porão de carga aberto contendo três containers modernos parcialmente encaixados, contrastando a era da navegação com a tecnologia atual.
- **Paleta de Cores (CSS/UI):**
  - Fundo principal: `#0B1320` (Azul-marinho profundo)
  - Elementos secundários: `#243447` (Azul aço)
  - Destaque/Ações: `#C49A45` (Dourado envelhecido)
  - Texto e Ícones: `#E8E2D0` (Marfim)
  - Alertas/Detalhes: `#7A263A` (Vermelho vinho)

## 2. Decisões Arquiteturais
- **Backend:** Go (Single binary).
- **Frontend:** HTML embutido (`embed.FS`), HTMX para reatividade, SSE para logs em tempo real. Estilização baseada na paleta definida.
- **Banco de Dados:** SQLite em modo WAL (apenas para configurações de painel, 2FA e auth).
- **Autenticação:** Dual-Realm (Linux PAM via SO ou Interno no SQLite) + TOTP (2FA).
- **Integração Docker:** Acesso direto ao socket (`/var/run/docker.sock`) usando o SDK oficial do Go (`moby/moby/client`).
- **Deploy (Futuro):** Empacotamento `.deb` usando nFPM, rodando como serviço systemd autônomo com geração automática de certificado TLS.
