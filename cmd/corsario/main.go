package main

import (
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"corsario/internal/auth"
	"corsario/internal/db"
	"corsario/internal/docker"
	"corsario/internal/ui"

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

type LoginData struct {
	Error string
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

	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
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
		if err := tmpl.ExecuteTemplate(w, "containers_table", data); err != nil {
			http.Error(w, "Erro ao renderizar tabela de containers", http.StatusInternalServerError)
		}
	}))

	// Rotas de controle de ciclo de vida dos containers protegidas pelo middleware de autenticação
	mux.HandleFunc("POST /containers/start/{id}", authMiddleware(database, handleContainerAction(docker.StartContainer)))
	mux.HandleFunc("POST /containers/stop/{id}", authMiddleware(database, handleContainerAction(docker.StopContainer)))
	mux.HandleFunc("POST /containers/restart/{id}", authMiddleware(database, handleContainerAction(docker.RestartContainer)))
	mux.HandleFunc("DELETE /containers/remove/{id}", authMiddleware(database, handleContainerAction(docker.RemoveContainer)))

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
