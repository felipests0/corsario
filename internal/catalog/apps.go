package catalog

type AppTemplate struct {
	Name         string
	Image        string
	DefaultPort  string
	RequiredEnvs []string
}

func GetTemplates() []AppTemplate {
	return []AppTemplate{
		{
			Name:         "PostgreSQL",
			Image:        "postgres:15-alpine",
			DefaultPort:  "5432:5432",
			RequiredEnvs: []string{"POSTGRES_PASSWORD="},
		},
		{
			Name:         "Nginx",
			Image:        "nginx:alpine",
			DefaultPort:  "80:80",
			RequiredEnvs: []string{},
		},
		{
			Name:         "Node.js",
			Image:        "node:20-alpine",
			DefaultPort:  "3000:3000",
			RequiredEnvs: []string{},
		},
	}
}
