package docker

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
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
	ContainerCreate(ctx context.Context, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *ocispec.Platform, containerName string) (container.CreateResponse, error)
	ContainerStart(ctx context.Context, containerID string, options container.StartOptions) error
	ContainerStop(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRestart(ctx context.Context, containerID string, options container.StopOptions) error
	ContainerRemove(ctx context.Context, containerID string, options container.RemoveOptions) error
	ContainerLogs(ctx context.Context, container string, options container.LogsOptions) (io.ReadCloser, error)
	ImageList(ctx context.Context, options image.ListOptions) ([]image.Summary, error)
	ImagePull(ctx context.Context, refStr string, options image.PullOptions) (io.ReadCloser, error)
	ImageRemove(ctx context.Context, imageID string, options image.RemoveOptions) ([]image.DeleteResponse, error)
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

func ListImages() ([]image.Summary, error) {
	cli, err := newContainerClient()
	if err != nil {
		return nil, err
	}
	defer cli.Close()

	images, err := cli.ImageList(context.Background(), image.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("erro ao listar imagens: %w", err)
	}

	for i := range images {
		shortID := strings.TrimPrefix(images[i].ID, "sha256:")
		if len(shortID) > 12 {
			shortID = shortID[:12]
		}
		images[i].ID = shortID
		if len(images[i].RepoTags) == 0 {
			images[i].RepoTags = []string{"<none>:<none>"}
		}
	}

	return images, nil
}

func RemoveImage(id string) error {
	cli, err := newContainerClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	if _, err := cli.ImageRemove(context.Background(), id, image.RemoveOptions{Force: true, PruneChildren: true}); err != nil {
		return fmt.Errorf("erro ao remover imagem %s: %w", id, err)
	}
	return nil
}

func CreateAndStartContainer(name, imageName, portMapping string, envs []string) error {
	cli, err := newContainerClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	ctx := context.Background()

	pullReader, err := cli.ImagePull(ctx, imageName, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("erro ao puxar imagem %s: %w", imageName, err)
	}
	_, _ = io.Copy(io.Discard, pullReader)
	_ = pullReader.Close()

	exposedPorts := nat.PortSet{}
	portBindings := nat.PortMap{}

	if strings.TrimSpace(portMapping) != "" {
		parts := strings.Split(strings.TrimSpace(portMapping), ":")
		var hostPort, containerPort string
		if len(parts) == 2 {
			hostPort = strings.TrimSpace(parts[0])
			containerPort = strings.TrimSpace(parts[1])
		} else {
			hostPort = strings.TrimSpace(parts[0])
			containerPort = strings.TrimSpace(parts[0])
		}
		if !strings.Contains(containerPort, "/") {
			containerPort += "/tcp"
		}
		p := nat.Port(containerPort)
		exposedPorts[p] = struct{}{}
		portBindings[p] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: hostPort,
			},
		}
	}

	config := &container.Config{
		Image:        imageName,
		Env:          envs,
		ExposedPorts: exposedPorts,
		Tty:          true,
	}

	hostConfig := &container.HostConfig{
		PortBindings: portBindings,
	}

	resp, err := cli.ContainerCreate(ctx, config, hostConfig, nil, nil, strings.TrimSpace(name))
	if err != nil {
		return fmt.Errorf("erro ao criar container %s: %w", name, err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("erro ao iniciar container %s: %w", resp.ID, err)
	}

	return nil
}
