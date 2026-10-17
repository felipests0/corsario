package docker

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
)

type mockContainerClient struct {
	startedID        string
	stoppedID        string
	restartedID      string
	removedID        string
	removeForce      bool
	logsID           string
	logsOptions      container.LogsOptions
	removedImageID   string
	removeImageForce bool
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

func (m *mockContainerClient) ContainerLogs(ctx context.Context, containerID string, options container.LogsOptions) (io.ReadCloser, error) {
	m.logsID = containerID
	m.logsOptions = options
	return io.NopCloser(strings.NewReader("2026-10-17T01:38:00Z log de teste\n")), nil
}

func (m *mockContainerClient) ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error) {
	return []image.Summary{
		{
			ID:       "sha256:1234567890abcdef1234567890abcdef",
			RepoTags: []string{"nginx:latest"},
			Size:     142606336,
		},
	}, nil
}

func (m *mockContainerClient) ImageRemove(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error) {
	m.removedImageID = imageID
	m.removeImageForce = options.Force
	return nil, nil
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

	rc, err := ContainerLogs("c-logs-123")
	if err != nil {
		t.Fatalf("erro inesperado em ContainerLogs: %v", err)
	}
	defer rc.Close()

	if mockCli.logsID != "c-logs-123" {
		t.Errorf("esperava logsID 'c-logs-123', obteve '%s'", mockCli.logsID)
	}
	if !mockCli.logsOptions.ShowStdout || !mockCli.logsOptions.ShowStderr || !mockCli.logsOptions.Follow || !mockCli.logsOptions.Timestamps || mockCli.logsOptions.Tail != "50" {
		t.Errorf("opções de logs incorretas: %+v", mockCli.logsOptions)
	}
}

func TestImageActionsWithMock(t *testing.T) {
	mockCli := &mockContainerClient{}
	origFactory := newContainerClient
	newContainerClient = func() (ContainerAPI, error) {
		return mockCli, nil
	}
	defer func() {
		newContainerClient = origFactory
	}()

	imgs, err := ListImages()
	if err != nil {
		t.Fatalf("erro inesperado em ListImages: %v", err)
	}
	if len(imgs) != 1 || imgs[0].ID != "1234567890ab" {
		t.Errorf("esperava 1 imagem com ID curto '1234567890ab', obteve %+v", imgs)
	}

	if err := RemoveImage("img-remove-123"); err != nil {
		t.Fatalf("erro inesperado em RemoveImage: %v", err)
	}
	if mockCli.removedImageID != "img-remove-123" {
		t.Errorf("esperava removedImageID 'img-remove-123', obteve '%s'", mockCli.removedImageID)
	}
	if !mockCli.removeImageForce {
		t.Error("esperava image.RemoveOptions{Force: true} em RemoveImage")
	}
}
