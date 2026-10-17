package docker

import (
	"context"
	"testing"

	"github.com/docker/docker/api/types/container"
)

type mockContainerClient struct {
	startedID   string
	stoppedID   string
	restartedID string
	removedID   string
	removeForce bool
}

func (m *mockContainerClient) ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error) {
	return nil, nil
}

func (m *mockContainerClient) ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error {
	m.startedID = containerID
	return nil
}

func (m *mockContainerClient) ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error {
	m.stoppedID = containerID
	return nil
}

func (m *mockContainerClient) ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error {
	m.restartedID = containerID
	return nil
}

func (m *mockContainerClient) ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error {
	m.removedID = containerID
	m.removeForce = options.Force
	return nil
}

func (m *mockContainerClient) Close() error {
	return nil
}

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

func TestContainerLifecycleActionsWithMock(t *testing.T) {
	mockCli := &mockContainerClient{}
	origFactory := newContainerClient
	newContainerClient = func() (ContainerAPI, error) {
		return mockCli, nil
	}
	defer func() {
		newContainerClient = origFactory
	}()

	if err := StartContainer("c-start-123"); err != nil {
		t.Fatalf("erro inesperado em StartContainer: %v", err)
	}
	if mockCli.startedID != "c-start-123" {
		t.Errorf("esperava startedID 'c-start-123', obteve '%s'", mockCli.startedID)
	}

	if err := StopContainer("c-stop-123"); err != nil {
		t.Fatalf("erro inesperado em StopContainer: %v", err)
	}
	if mockCli.stoppedID != "c-stop-123" {
		t.Errorf("esperava stoppedID 'c-stop-123', obteve '%s'", mockCli.stoppedID)
	}

	if err := RestartContainer("c-restart-123"); err != nil {
		t.Fatalf("erro inesperado em RestartContainer: %v", err)
	}
	if mockCli.restartedID != "c-restart-123" {
		t.Errorf("esperava restartedID 'c-restart-123', obteve '%s'", mockCli.restartedID)
	}

	if err := RemoveContainer("c-remove-123"); err != nil {
		t.Fatalf("erro inesperado em RemoveContainer: %v", err)
	}
	if mockCli.removedID != "c-remove-123" {
		t.Errorf("esperava removedID 'c-remove-123', obteve '%s'", mockCli.removedID)
	}
	if !mockCli.removeForce {
		t.Error("esperava RemoveOptions{Force: true} em RemoveContainer")
	}
}
