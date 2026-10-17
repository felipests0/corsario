package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"

	"corsario/internal/db"
	"corsario/internal/docker"
	"corsario/internal/ui"

	_ "github.com/mattn/go-sqlite3"
)

var (
	staticFS    embed.FS = ui.StaticFS
	templatesFS embed.FS = ui.TemplatesFS
)

type DashboardData struct {
	Containers []docker.ContainerInfo
	Error      string
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

func main() {
	database, err := db.InitDB("corsario.db")
	if err != nil {
		log.Fatalf("Erro ao inicializar banco de dados: %v", err)
	}
	defer database.Close()

	tmpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Erro ao carregar templates: %v", err)
	}

	mux := http.NewServeMux()

	// Servir arquivos estáticos em /static/
	mux.Handle("/static/", http.FileServer(http.FS(staticFS)))

	// Rota principal / com o dashboard como aba principal
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data := loadDashboardData()
		if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
			http.Error(w, "Erro ao renderizar página", http.StatusInternalServerError)
		}
	})

	// Rota GET /containers que retorna o fragmento HTML da tabela para o HTMX
	mux.HandleFunc("/containers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Método não permitido", http.StatusMethodNotAllowed)
			return
		}
		data := loadDashboardData()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.ExecuteTemplate(w, "containers_table", data); err != nil {
			http.Error(w, "Erro ao renderizar tabela de containers", http.StatusInternalServerError)
		}
	})

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
