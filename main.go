package main

import (
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	bot         *Telegram
	superAdmins = []int64{}
	docker      *Docker
)

func main() {
	// Configure logger
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if os.Getenv("TELEDOCK_DEBUG") == "true" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	log.Logger = log.With().Caller().Logger()

	//Check required env vars

	for _, envVar := range []string{
		"TELEDOCK_TOKEN",
		"TELEDOCK_SUPERADMINS",
	} {
		if os.Getenv(envVar) == "" {
			log.Fatal().Str("module", "main").Str("envvar", envVar).Msg("missing environment variable")
		}
	}

	// Load superadmin list
	if envsa := os.Getenv("TELEDOCK_SUPERADMINS"); envsa != "" {
		for _, uidStr := range strings.Split(envsa, ",") {
			uid, errp := parseInt64(uidStr)

			if errp != nil {
				log.Fatal().Str("module", "main").Err(errp).Msg("failed parsing superadmins list")
			}

			superAdmins = append(superAdmins, uid)
		}
	}
	log.Info().Str("module", "main").Ints64("user_ids", superAdmins).Msg("loaded superadmins")

	// Connect to docker
	var err error
	docker, err = newDocker()
	if err != nil {
		log.Fatal().Str("module", "main").Err(err).Msg("failed to connect to docker")
	}

	// Ping the daemon
	docker.ping()

	// Create bot
	bot, err = NewBot(os.Getenv("TELEDOCK_TOKEN"))

	if err != nil {
		log.Fatal().Str("module", "main").Err(err).Msg("failed bot instantiaion")
	}

	// Start the bot
	bot.Start()
}
