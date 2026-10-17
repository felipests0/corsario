package catalog

import "testing"

func TestGetTemplates(t *testing.T) {
	templates := GetTemplates()
	if len(templates) != 9 {
		t.Fatalf("esperava 9 templates no catálogo, obteve %d", len(templates))
	}

	expectedNames := []string{
		"PostgreSQL",
		"Nginx",
		"Node.js",
		"Redis",
		"MySQL",
		"MongoDB",
		"WordPress",
		"Zabbix",
		"Wazuh",
	}

	found := make(map[string]bool)
	for _, tmpl := range templates {
		found[tmpl.Name] = true
		if tmpl.Image == "" || tmpl.DefaultPort == "" {
			t.Errorf("template %s possui Image ou DefaultPort vazio", tmpl.Name)
		}
	}

	for _, name := range expectedNames {
		if !found[name] {
			t.Errorf("esperava encontrar template '%s' no catálogo", name)
		}
	}
}
