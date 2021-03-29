package main

import (
	"context"
	"io"
	"regexp"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/rs/zerolog/log"
)

// Docker represents a docker client
type Docker struct {
	cli *client.Client
	ctx context.Context
}

func newDocker() (*Docker, error) {
	cli, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	log.Info().Str("module", "docker").Msg("connected to the docker daemon")
	return &Docker{cli: cli, ctx: ctx}, nil
}

func (d *Docker) ping() {
	ping, err := d.cli.Ping(d.ctx)

	if err != nil {
		log.Fatal().Str("module", "docker").Err(err).Msg("error pinging docker")
	}
	log.Info().Str("module", "docker").Str("Api-version", ping.APIVersion).Msg("docker daemon health check")
}

func (d *Docker) list(options types.ContainerListOptions) []types.Container {
	containers, err := d.cli.ContainerList(d.ctx, options)
	if err != nil {
		log.Fatal().Str("module", "docker").Err(err).Msg("error retrieving containers")
	}
	return containers
}

func (d *Docker) stop(containerID string) error {
	timeout := time.Until(time.Now().Add(30 * time.Second))
	if err := d.cli.ContainerStop(d.ctx, containerID, &timeout); err != nil {
		log.Fatal().Str("module", "docker").Str("containerID", containerID).Err(err).Msg("error stoping container")
		return err
	}
	return nil
}

func (d *Docker) start(containerID string) error {
	if err := d.cli.ContainerStart(d.ctx, containerID, types.ContainerStartOptions{}); err != nil {
		log.Fatal().Str("module", "docker").Str("containerID", containerID).Err(err).Msg("error starting container")
		return err
	}
	return nil
}

func (d *Docker) inspect(containerID string) (*types.ContainerJSON, error) {
	container, err := d.cli.ContainerInspect(d.ctx, containerID)
	if err != nil {
		log.Fatal().Str("module", "docker").Str("containerID", containerID).Err(err).Msg("error inspecting container")
		return nil, err
	}
	return &container, nil
}

func (d *Docker) logs(containerID string, tail string) ([]string, error) {
	var (
		bytes []byte   = make([]byte, 3000) // Telegram message length limit
		logs  []string = nil
	)
	if tail != "all" && !isNumber(tail) {
		tail = "10"
	}
	logsReader, err := d.cli.ContainerLogs(d.ctx, containerID, types.ContainerLogsOptions{Tail: tail, ShowStderr: true, ShowStdout: true})
	if err != nil {
		log.Fatal().Str("module", "docker").Str("containerID", containerID).Err(err).Msg("error getting container logs")
		return nil, err
	}
	defer func() {
		err := logsReader.Close()
		if err != nil {
			log.Fatal().Str("module", "docker").Str("containerID", containerID).Err(err).Msg("error closing io.Reader")
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

func (d *Docker) isValidID(containerID string) bool {
	re := regexp.MustCompile(`(?m)^[A-Fa-f0-9]{10,12}$`)
	return re.MatchString(containerID)
}
