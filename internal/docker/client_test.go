package docker

import "testing"

func TestNewClient(t *testing.T) {
	cli, err := NewClient()
	if err != nil {
		t.Fatalf("esperava criação do cliente Docker sem erros, mas obteve: %v", err)
	}
	if cli == nil {
		t.Fatal("esperava cliente Docker não nulo (non-nil)")
	}
	defer cli.Close()
}
