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

## 3. Qualidade e Testes
- **Testes Unitários:** Todas as regras de negócio do backend, manipulação de banco de dados e integrações (como o Docker SDK) devem ser cobertas por testes unitários usando o pacote nativo `testing` do Go.
- **Banco de Dados em Testes:** Testes que envolvem o SQLite devem utilizar bancos em memória (`file::memory:?cache=shared`) para não poluir o banco de desenvolvimento.

## 4. Progresso Atual e Funcionalidades Implementadas
- Estrutura base com Go e `embed.FS` funcional.
- UI com HTMX e paleta de cores temática.
- Banco de dados SQLite operando em modo WAL (`internal/db`).
- Autenticação local, hash de senhas e middleware de controlo de sessões via cookies (`internal/auth`).
- Criação de utilizador admin via CLI (flags).
- Ligação direta ao `/var/run/docker.sock` via Docker SDK.
- Ações de ciclo de vida de containers operacionais (Start, Stop, Restart, Remove).
- Visualização de logs de containers em tempo real utilizando Server-Sent Events (SSE) e HTMX.
- Gestão do Arsenal (Listagem e exclusão forçada de imagens Docker).
- Estaleiro de Aplicações: Catálogo de templates pré-configurados e lançamento rápido de containers via formulários dinâmicos.
- UX: Feedback visual dinâmico (animação CSS) durante operações demoradas (como pull de imagens) usando o estado `htmx-request`.
- UI/UX: Integração da logo do Corsário no cabeçalho com animação reativa ao scroll (navio a "navegar").
- Neuro-UX: Fundo interativo reativo ao rato (efeito sonar no oceano) e micro-interações de flutuação nos cards para feedback visual imediato.
- UI/UX: Logo fixada à esquerda do título e adição de feixes de luz animados no background para maior profundidade.
- UI/UX: Configuração do favicon da aplicação utilizando a logo do navio.
- Gestão de Tripulação (CRUD de utilizadores locais no SQLite integrado na interface via HTMX).
