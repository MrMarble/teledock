package main

import (
	"flag"
	"os"
	"strings"
	"time"

	"github.com/mrmarble/teledock/internal/docker"
	"github.com/mrmarble/teledock/internal/telegram"
	"github.com/mrmarble/teledock/internal/utils"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Flags.
var (
	debug  = flag.Bool("debug", false, "enable debug log level")
	pretty = flag.Bool("pretty", false, "enable pretty logging (human-friendly)")
)

var (
	bot         *telegram.Telegram
	superAdmins = []int64{}
	dockr       *docker.Docker
)

func init() {
	flag.Parse()

	// Set logger preferences
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if *debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	if *pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	}
}

func main() {
	log.Info().Str("log_level", zerolog.GlobalLevel().String()).Msg("Starting BOT...")

	//Check required env vars

	for _, envVar := range []string{
		"TELEDOCK_TOKEN",
		"TELEDOCK_SUPERADMINS",
	} {
		if os.Getenv(envVar) == "" {
			log.Fatal().Str("envvar", envVar).Msg("missing environment variable")
		}
	}

	// Load superadmin list
	if envsa := os.Getenv("TELEDOCK_SUPERADMINS"); envsa != "" {
		for _, uidStr := range strings.Split(envsa, ",") {
			uid, errp := utils.ParseInt64(uidStr)

			if errp != nil {
				log.Fatal().Err(errp).Msg("failed parsing superadmins list")
			}

			superAdmins = append(superAdmins, uid)
		}
	}
	log.Info().Ints64("user_ids", superAdmins).Msg("loaded superadmins")

	// Connect to docker
	var err error
	dockr, err = docker.NewDocker()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to docker")
	}

	// Ping the daemon
	dockr.Ping()

	// Create bot
	bot, err = telegram.NewBot(os.Getenv("TELEDOCK_TOKEN"), *dockr, superAdmins)

	if err != nil {
		log.Fatal().Err(err).Msg("failed bot instantiaion")
	}

	// Start the bot
	bot.Start()
}
