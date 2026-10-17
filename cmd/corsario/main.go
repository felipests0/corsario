package main

import (
	"bufio"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"html"
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"corsario/internal/auth"
	"corsario/internal/catalog"
	"corsario/internal/db"
	"corsario/internal/docker"
	"corsario/internal/ui"

	"github.com/docker/docker/api/types/image"
	_ "github.com/mattn/go-sqlite3"
)

var (
	staticFS    embed.FS = ui.StaticFS
	templatesFS embed.FS = ui.TemplatesFS
)

const sessionCookieName = "corsario_session"

type DashboardData struct {
	Containers []docker.ContainerInfo
	Error      string
}

type ImagesData struct {
	Images []image.Summary
	Error  string
}

type CatalogData struct {
	Templates []catalog.AppTemplate
}

type LoginData struct {
	Error string
}

type LogsBoxData struct {
	ID string
}

func loadDashboardData() DashboardData {
	containers, err := docker.ListContainers()
	if err != nil {
		return DashboardData{
			Error: fmt.Sprintf("Erro ao consultar o Docker: %v", err),
		}
	}
	return DashboardData{
		Containers: containers,
	}
}

func loadImagesData() ImagesData {
	images, err := docker.ListImages()
	if err != nil {
		return ImagesData{
			Error: fmt.Sprintf("Erro ao listar imagens do Docker: %v", err),
		}
	}
	return ImagesData{
		Images: images,
	}
}

func authMiddleware(database *sql.DB, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(sessionCookieName)
		if err != nil || cookie.Value == "" {
			redirectToLogin(w, r)
			return
		}

		if _, valid := db.ValidateSession(database, cookie.Value); !valid {
			redirectToLogin(w, r)
			return
		}

		next(w, r)
	}
}

func redirectToLogin(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func handleContainerAction(action func(id string) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "ID do container ausente", http.StatusBadRequest)
			return
		}
		if err := action(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("HX-Trigger", "containerListChanged")
		w.WriteHeader(http.StatusOK)
	}
}

func sanitizeDockerLogLine(line string) string {
	// Remove cabeçalho de multiplexação de 8 bytes do Docker (se presente)
	if len(line) >= 8 && (line[0] == 1 || line[0] == 2) && line[1] == 0 && line[2] == 0 && line[3] == 0 {
		line = line[8:]
	}
	return strings.TrimRight(line, "\r\n")
}

func main() {
	createAdmin := flag.String("create-admin", "", "Username do administrador inicial a ser criado")
	password := flag.String("password", "", "Senha do administrador inicial")
	flag.Parse()

	database, err := db.InitDB("corsario.db")
	if err != nil {
		log.Fatalf("Erro ao inicializar banco de dados: %v", err)
	}
	defer database.Close()

	if *createAdmin != "" || *password != "" {
		if *createAdmin == "" || *password == "" {
			log.Fatal("Ambos os parâmetros -create-admin e -password devem ser informados.")
		}
		hash, err := auth.HashPassword(*password)
		if err != nil {
			log.Fatalf("Erro ao gerar hash da senha: %v", err)
		}
		if err := db.CreateAdminUser(database, *createAdmin, hash); err != nil {
			log.Fatalf("Erro ao criar usuário administrador: %v", err)
		}
		fmt.Printf("Usuário administrador '%s' (realm: interno) criado com sucesso!\n", *createAdmin)
		os.Exit(0)
	}

	funcMap := template.FuncMap{
		"sizeMB": func(bytes int64) string {
			return fmt.Sprintf("%.2f MB", float64(bytes)/(1024.0*1024.0))
		},
		"joinTags": func(tags []string) string {
			if len(tags) == 0 {
				return "<none>:<none>"
			}
			return strings.Join(tags, ", ")
		},
		"envKey": func(env string) string {
			return strings.TrimSuffix(env, "=")
		},
	}

	tmpl, err := template.New("").Funcs(funcMap).ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Erro ao carregar templates: %v", err)
	}

	mux := http.NewServeMux()

	// Servir arquivos estáticos em /static/
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	// Rota /login (GET e POST)
	mux.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := tmpl.ExecuteTemplate(w, "login.html", LoginData{}); err != nil {
				http.Error(w, "Erro ao renderizar tela de login", http.StatusInternalServerError)
			}

		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_ = tmpl.ExecuteTemplate(w, "login.html", LoginData{Error: "Dados de formulário inválidos."})
				return
			}

			username := r.FormValue("username")
			pass := r.FormValue("password")
			realm := r.FormValue("realm")
			if realm == "" {
				realm = "interno"
			}

			user, err := db.GetUserByUsernameAndRealm(database, username, realm)
			if err != nil || !auth.CheckPasswordHash(pass, user.PasswordHash) {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_ = tmpl.ExecuteTemplate(w, "login.html", LoginData{
					Error: "Credenciais inválidas. Verifique seu usuário, senha e realm.",
				})
				return
			}

			token, err := auth.GenerateSessionToken()
			if err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_ = tmpl.ExecuteTemplate(w, "login.html", LoginData{
					Error: "Erro interno ao gerar sessão.",
				})
				return
			}

			expiresAt := time.Now().UTC().Add(24 * time.Hour)
			if err := db.CreateSession(database, token, user.ID, expiresAt); err != nil {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_ = tmpl.ExecuteTemplate(w, "login.html", LoginData{
					Error: "Erro ao registrar sessão no banco de dados.",
				})
				return
			}

			http.SetCookie(w, &http.Cookie{
				Name:     sessionCookieName,
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				Expires:  expiresAt,
				MaxAge:   86400,
				SameSite: http.SameSiteLaxMode,
			})

			if r.Header.Get("HX-Request") == "true" {
				w.Header().Set("HX-Redirect", "/")
				w.WriteHeader(http.StatusOK)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)

		default:
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
		}
	})

	// Rota principal / protegida pelo middleware de autenticação
	mux.HandleFunc("/", authMiddleware(database, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := loadDashboardData()
		if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
			http.Error(w, "Erro ao renderizar página", http.StatusInternalServerError)
		}
	}))

	// Rota GET /containers protegida pelo middleware de autenticação
	mux.HandleFunc("/containers", authMiddleware(database, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}
		data := loadDashboardData()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		target := r.Header.Get("HX-Target")
		templateName := "containers_table"
		if target == "conteudo-principal" {
			templateName = "dashboard"
		}
		if err := tmpl.ExecuteTemplate(w, templateName, data); err != nil {
			http.Error(w, "Erro ao renderizar tabela de containers", http.StatusInternalServerError)
		}
	}))

	// Rota GET /catalog protegida pelo middleware de autenticação
	mux.HandleFunc("GET /catalog", authMiddleware(database, func(w http.ResponseWriter, r *http.Request) {
		data := CatalogData{
			Templates: catalog.GetTemplates(),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "catalog.html", data); err != nil {
			http.Error(w, "Erro ao renderizar estaleiro", http.StatusInternalServerError)
		}
	}))

	// Rota POST /containers/create protegida pelo middleware de autenticação
	mux.HandleFunc("POST /containers/create", authMiddleware(database, func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, `<div class="alert-box">Dados de formulário inválidos.</div>`)
			return
		}

		name := r.FormValue("name")
		img := r.FormValue("image")
		port := r.FormValue("port")

		var envs []string
		for k, vals := range r.PostForm {
			if strings.HasPrefix(k, "env_") && len(vals) > 0 {
				envName := strings.TrimPrefix(k, "env_")
				envs = append(envs, fmt.Sprintf("%s=%s", envName, vals[0]))
			}
		}

		if err := docker.CreateAndStartContainer(name, img, port, envs); err != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, `<div class="alert-box">Erro ao lançar container: %s</div>`, html.EscapeString(err.Error()))
			return
		}

		w.Header().Set("HX-Redirect", "/")
		w.WriteHeader(http.StatusOK)
	}))

	// Rotas de controle de ciclo de vida dos containers protegidas pelo middleware de autenticação
	mux.HandleFunc("POST /containers/start/{id}", authMiddleware(database, handleContainerAction(docker.StartContainer)))
	mux.HandleFunc("POST /containers/stop/{id}", authMiddleware(database, handleContainerAction(docker.StopContainer)))
	mux.HandleFunc("POST /containers/restart/{id}", authMiddleware(database, handleContainerAction(docker.RestartContainer)))
	mux.HandleFunc("DELETE /containers/remove/{id}", authMiddleware(database, handleContainerAction(docker.RemoveContainer)))

	// Rota GET /images protegida pelo middleware de autenticação
	mux.HandleFunc("GET /images", authMiddleware(database, func(w http.ResponseWriter, r *http.Request) {
		data := loadImagesData()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "images.html", data); err != nil {
			http.Error(w, "Erro ao renderizar lista de imagens", http.StatusInternalServerError)
		}
	}))

	// Rota DELETE /images/remove/{id} protegida pelo middleware de autenticação
	mux.HandleFunc("DELETE /images/remove/{id}", authMiddleware(database, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "ID da imagem ausente", http.StatusBadRequest)
			return
		}
		if err := docker.RemoveImage(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("HX-Trigger", "imageListChanged")
		w.WriteHeader(http.StatusOK)
	}))

	// Rota GET /containers/logs-box/{id} para exibir o painel SSE de logs do container
	mux.HandleFunc("GET /containers/logs-box/{id}", authMiddleware(database, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "ID do container ausente", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "logs_box", LogsBoxData{ID: id}); err != nil {
			http.Error(w, "Erro ao renderizar caixa de logs", http.StatusInternalServerError)
		}
	}))

	// Rota SSE GET /containers/logs/{id} para streaming de logs em tempo real
	mux.HandleFunc("GET /containers/logs/{id}", authMiddleware(database, func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "ID do container ausente", http.StatusBadRequest)
			return
		}

		f, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming SSE não suportado", http.StatusInternalServerError)
			return
		}

		logStream, err := docker.ContainerLogs(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer logStream.Close()

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		go func() {
			<-r.Context().Done()
			_ = logStream.Close()
		}()

		scanner := bufio.NewScanner(logStream)
		for scanner.Scan() {
			select {
			case <-r.Context().Done():
				return
			default:
				linha := fmt.Sprintf("<span class=\"log-line\">%s</span>", html.EscapeString(sanitizeDockerLogLine(scanner.Text())))
				fmt.Fprintf(w, "data: %s\n\n", linha)
				f.Flush()
			}
		}
	}))

	// Rota /ping para HTMX
	mux.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, "<p>Pong do servidor!</p>")
	})

	addr := ":9443"
	fmt.Printf("Corsário: O navio está ancorado na porta %s e a tripulação a postos.\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Erro no servidor HTTP: %v", err)
	}
}
