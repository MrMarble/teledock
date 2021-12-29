package docker

import (
	"context"
	"io"
	"regexp"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/mrmarble/teledock/internal/constants"
	"github.com/mrmarble/teledock/internal/utils"

	"github.com/rs/zerolog"
	zero "github.com/rs/zerolog/log"
)

var log zerolog.Logger

// Docker represents a docker client.
type Docker struct {
	cli *client.Client
	ctx context.Context
}

func NewDocker() (*Docker, error) {
	log = zero.With().Str("package", "Docker").Logger()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	log.Info().Msg("connected to the docker daemon")
	return &Docker{cli, ctx}, nil
}

func (d *Docker) Ping() {
	ping, err := d.cli.Ping(d.ctx)

	if err != nil {
		log.Fatal().Err(err).Msg("error pinging docker")
	}
	log.Info().Str("Api-version", ping.APIVersion).Msg("docker daemon health check")
}

func (d *Docker) List(options types.ContainerListOptions) []types.Container {
	containers, err := d.cli.ContainerList(d.ctx, options)
	if err != nil {
		log.Fatal().Err(err).Msg("error retrieving containers")
	}
	return containers
}

func (d *Docker) ListImages(options types.ImageListOptions) []types.ImageSummary {
	images, err := d.cli.ImageList(d.ctx, options)
	if err != nil {
		log.Fatal().Err(err).Msg("error retrieving images")
	}
	return images
}

func (d *Docker) ListCompose() map[string][]types.Container {
	var (
		filters = filters.NewArgs()
		stacks  = map[string][]types.Container{}
	)
	filters.Add("label", constants.ComposeLabel)
	containers := d.List(types.ContainerListOptions{All: true, Filters: filters})
	for _, container := range containers {
		stacks[container.Labels[constants.ComposeLabel]] = append(stacks[container.Labels[constants.ComposeLabel]], container)
	}
	return stacks
}

func (d *Docker) Stop(containerID string) error {
	timeout := time.Until(time.Now().Add(30 * time.Second))
	if err := d.cli.ContainerStop(d.ctx, containerID, &timeout); err != nil {
		log.Fatal().Str("containerID", containerID).Err(err).Msg("error stoping container")
		return err
	}
	return nil
}

func (d *Docker) Start(containerID string) error {
	if err := d.cli.ContainerStart(d.ctx, containerID, types.ContainerStartOptions{}); err != nil {
		log.Fatal().Str("containerID", containerID).Err(err).Msg("error starting container")
		return err
	}
	return nil
}

func (d *Docker) Inspect(containerID string) (*types.ContainerJSON, error) {
	container, err := d.cli.ContainerInspect(d.ctx, containerID)
	if err != nil {
		log.Fatal().Str("containerID", containerID).Err(err).Msg("error inspecting container")
		return nil, err
	}
	return &container, nil
}

func (d *Docker) Logs(containerID string, tail string) ([]string, error) {
	var (
		bytes = make([]byte, 3000) // Telegram message length limit
		logs  []string
	)
	if tail != "all" && !utils.IsNumber(tail) {
		tail = "10"
	}
	logsReader, err := d.cli.ContainerLogs(d.ctx, containerID, types.ContainerLogsOptions{Tail: tail, ShowStderr: true, ShowStdout: true})
	if err != nil {
		log.Fatal().Str("containerID", containerID).Err(err).Msg("error getting container logs")
		return nil, err
	}
	defer func() {
		err := logsReader.Close()
		if err != nil {
			log.Fatal().Str("containerID", containerID).Err(err).Msg("error closing io.Reader")
		}
	}()

	for {
		numBytes, err := logsReader.Read(bytes)
		logs = append(logs, string(bytes[:numBytes]))
		if err == io.EOF {
			break
		}
	}
	return logs, nil
}

func (d *Docker) IsValidID(containerID string) bool {
	re := regexp.MustCompile(`(?m)^[A-Fa-f0-9]{10,12}$`)
	return re.MatchString(containerID)
}
