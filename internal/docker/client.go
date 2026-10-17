package docker

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type ContainerInfo struct {
	ID     string
	Name   string
	Image  string
	State  string
	Status string
	Ports  string
}

type ContainerAPI interface {
	ContainerList(ctx context.Context, options container.ListOptions) ([]container.Summary, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerLogs(ctx context.Context, container string, options container.LogsOptions) (io.ReadCloser, error)
	Close() error
}

var newContainerClient = func() (ContainerAPI, error) {
	return NewClient()
}

type logStreamCloser struct {
	io.ReadCloser
	cli ContainerAPI
}

func (l *logStreamCloser) Close() error {
	err := l.ReadCloser.Close()
	_ = l.cli.Close()
	return err
}

func NewClient() (*client.Client, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost("unix:///var/run/docker.sock"),
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("erro ao criar cliente Docker: %w", err)
	}
	return cli, nil
}

func ListContainers() ([]ContainerInfo, error) {
	cli, err := newContainerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	containers, err := cli.ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("erro ao listar containers: %w", err)
	}

	result := make([]ContainerInfo, 0, len(containers))
	for _, c := range containers {
		shortID := c.ID
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}

		name := "-"
		if len(c.Names) > 0 {
			name = strings.TrimPrefix(c.Names[0], "/")
		}

		portStrs := make([]string, 0, len(c.Ports))
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				ip := p.IP
				if ip == "" {
					ip = "0.0.0.0"
				}
				portStrs = append(portStrs, fmt.Sprintf("%s:%d->%d/%s", ip, p.PublicPort, p.PrivatePort, p.Type))
			} else {
				portStrs = append(portStrs, fmt.Sprintf("%d/%s", p.PrivatePort, p.Type))
			}
		}
		ports := "-"
		if len(portStrs) > 0 {
			ports = strings.Join(portStrs, ", ")
		}

		result = append(result, ContainerInfo{
			ID:     shortID,
			Name:   name,
			Image:  c.Image,
			State:  c.State,
			Status: c.Status,
			Ports:  ports,
		})
	}

	return result, nil
}

func StartContainer(id string) error {
	cli, err := newContainerClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if err := cli.ContainerStart(context.Background(), id, container.StartOptions{}); err != nil {
		return fmt.Errorf("erro ao iniciar container %s: %w", id, err)
	}
	return nil
}

func StopContainer(id string) error {
	cli, err := newContainerClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if err := cli.ContainerStop(context.Background(), id, container.StopOptions{}); err != nil {
		return fmt.Errorf("erro ao parar container %s: %w", id, err)
	}
	return nil
}

func RestartContainer(id string) error {
	cli, err := newContainerClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if err := cli.ContainerRestart(context.Background(), id, container.StopOptions{}); err != nil {
		return fmt.Errorf("erro ao reiniciar container %s: %w", id, err)
	}
	return nil
}

func RemoveContainer(id string) error {
	cli, err := newContainerClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if err := cli.ContainerRemove(context.Background(), id, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("erro ao remover container %s: %w", id, err)
	}
	return nil
}

func ContainerLogs(id string) (io.ReadCloser, error) {
	cli, err := newContainerClient()
	if err != nil {
		return nil, err
	}

	opts := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
		Tail:       "50",
	}

	stream, err := cli.ContainerLogs(context.Background(), id, opts)
	if err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("erro ao obter logs do container %s: %w", id, err)
	}

	return &logStreamCloser{
		ReadCloser: stream,
		cli:        cli,
	}, nil
}
