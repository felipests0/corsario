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
		{
			Name:         "Redis",
			Image:        "redis:7-alpine",
			DefaultPort:  "6379:6379",
			RequiredEnvs: []string{},
		},
		{
			Name:         "MySQL",
			Image:        "mysql:8.0",
			DefaultPort:  "3306:3306",
			RequiredEnvs: []string{"MYSQL_ROOT_PASSWORD="},
		},
		{
			Name:         "MongoDB",
			Image:        "mongo:7.0",
			DefaultPort:  "27017:27017",
			RequiredEnvs: []string{"MONGO_INITDB_ROOT_USERNAME=", "MONGO_INITDB_ROOT_PASSWORD="},
		},
		{
			Name:         "WordPress",
			Image:        "wordpress:latest",
			DefaultPort:  "8080:80",
			RequiredEnvs: []string{"WORDPRESS_DB_HOST=", "WORDPRESS_DB_PASSWORD="},
		},
		{
			Name:         "Zabbix",
			Image:        "zabbix/zabbix-appliance:alpine-latest",
			DefaultPort:  "8081:8080",
			RequiredEnvs: []string{},
		},
		{
			Name:         "Wazuh",
			Image:        "wazuh/wazuh-manager:4.9.0",
			DefaultPort:  "55000:55000",
			RequiredEnvs: []string{},
		},
	}
}
