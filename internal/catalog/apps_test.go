package catalog

import "testing"

func TestGetTemplates(t *testing.T) {
	templates := GetTemplates()
	if len(templates) != 3 {
		t.Fatalf("esperava 3 templates no catálogo, obteve %d", len(templates))
	}

	expected := map[string]struct {
		image string
		port  string
		envs  int
	}{
		"PostgreSQL": {image: "postgres:15-alpine", port: "5432:5432", envs: 1},
		"Nginx":      {image: "nginx:alpine", port: "80:80", envs: 0},
		"Node.js":    {image: "node:20-alpine", port: "3000:3000", envs: 0},
	}

	for _, tmpl := range templates {
		exp, ok := expected[tmpl.Name]
		if !ok {
			t.Errorf("template inesperado no catálogo: %s", tmpl.Name)
			continue
		}
		if tmpl.Image != exp.image {
			t.Errorf("para %s esperava imagem %s, obteve %s", tmpl.Name, exp.image, tmpl.Image)
		}
		if tmpl.DefaultPort != exp.port {
			t.Errorf("para %s esperava porta %s, obteve %s", tmpl.Name, exp.port, tmpl.DefaultPort)
		}
		if len(tmpl.RequiredEnvs) != exp.envs {
			t.Errorf("para %s esperava %d envs, obteve %d", tmpl.Name, exp.envs, len(tmpl.RequiredEnvs))
		}
	}
}
