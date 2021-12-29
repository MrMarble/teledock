package main

import (
	"fmt"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/rs/zerolog/log"
)

// Telegram represents the telegram bot.
type Telegram struct {
	bot                *tb.Bot
	handlersRegistered bool
}

// Command represent a telegram command.
type Command struct {
	Cmd         string
	Aliases     []string
	Description string
	Handler     interface{}
}

// NewBot returns a Telegram bot.
func NewBot(token string) (*Telegram, error) {
	bot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		Reporter: func(err error) {
			log.Error().Str("module", "telegram").Err(err).Msg("telebot internal error")
		},
	})

	if err != nil {
		return nil, err
	}

	log.Info().Str("module", "telegram").Int64("id", bot.Me.ID).Str("name", bot.Me.FirstName).Str("username", bot.Me.Username).Msg("connected to telegram")

	return &Telegram{bot: bot}, nil
}

// Start starts polling for telegram updates.
func (t *Telegram) Start() {
	t.registerHandlers()

	log.Info().Str("module", "telegram").Msg("start polling")
	t.bot.Start()
}

// RegisterHandlers registers all the handlers.
func (t *Telegram) registerHandlers() {
	if t.handlersRegistered {
		return
	}

	log.Info().Str("module", "telegram").Msg("registering handlers")

	// Temporal list to send the commands to telegram
	botCommandList := []tb.Command{}
	var botCommands = []Command{
		{
			Handler:     t.handleStart,
			Cmd:         "start",
			Description: "Shows info",
		},
		{
			Handler:     t.handleList,
			Cmd:         "ps",
			Aliases:     []string{"ls", "list"},
			Description: "List running containers",
		},
		{
			Handler:     t.handleListAll,
			Cmd:         "psa",
			Aliases:     []string{"lsa", "listall"},
			Description: "List all containers",
		},
		{
			Handler:     t.handleStop,
			Cmd:         "stop",
			Aliases:     []string{"down"},
			Description: "Stop a running container. <ContainerID>",
		},
		{
			Handler:     t.handleStartContainer,
			Cmd:         "run",
			Description: "Start a stopped container. <ContainerID>",
		},
		{
			Handler:     t.handleInspect,
			Cmd:         "inspect",
			Aliases:     []string{"describe"},
			Description: "Inspect a container. <ContainerID>",
		},
		{
			Handler:     t.handleStacks,
			Cmd:         "stacks",
			Aliases:     []string{"lss", "liststacks"},
			Description: "Lists all compose stacks",
		},
		{
			Handler:     t.handleLogs,
			Cmd:         "logs",
			Description: "Shows container logs. <ContainerID> <tail>",
		},
		{
			Handler:     t.handleImageList,
			Cmd:         "images",
			Description: "List all installed images",
		},
	}
	for _, command := range botCommands {
		botCommandList = append(botCommandList, tb.Command{
			Text:        command.Cmd,
			Description: command.Description,
		})
		for _, alias := range command.Aliases {
			t.bot.Handle(fmt.Sprintf("/%s", alias), command.Handler)
		}
		t.bot.Handle(fmt.Sprintf("/%s", command.Cmd), command.Handler)
	}

	t.bot.Handle(tb.OnCallback, t.handleCallback)

	if err := t.bot.SetCommands(botCommandList); err != nil {
		log.Fatal().Str("module", "telegram").Err(err).Msg("error registering commands")
	}

	t.handlersRegistered = true
}

func (t *Telegram) isSuperAdmin(user *tb.User) bool {
	for _, uid := range superAdmins {
		if user.ID == uid {
			return true
		}
	}

	return false
}

// send sends a message with error logging and retries.
func (t *Telegram) send(to tb.Recipient, what interface{}, options ...interface{}) *tb.Message {
	hasParseMode := false
	for _, opt := range options {
		if _, hasParseMode = opt.(tb.ParseMode); hasParseMode {
			break
		}
	}

	if !hasParseMode {
		options = append(options, tb.ModeHTML)
	}

	try := 1
	for {
		msg, err := t.bot.Send(to, what, options...)

		if err == nil {
			return msg
		}

		if try > 5 {
			log.Error().Str("module", "telegram").Err(err).Msg("send aborted, retry limit exceeded")
			return nil
		}

		backoff := time.Second * 5 * time.Duration(try)
		log.Warn().Str("module", "telegram").Err(err).Str("sleep", backoff.String()).Msg("send failed, sleeping and retrying")
		time.Sleep(backoff)
		try++
	}
}

// reply replies a message with error logging and retries.
func (t *Telegram) reply(to *tb.Message, what interface{}, options ...interface{}) *tb.Message {
	hasParseMode := false
	for _, opt := range options {
		if _, hasParseMode = opt.(tb.ParseMode); hasParseMode {
			break
		}
	}

	if !hasParseMode {
		options = append(options, tb.ModeHTML)
	}

	try := 1
	for {
		msg, err := t.bot.Reply(to, what, options...)

		if err == nil {
			return msg
		}

		if try > 5 {
			log.Error().Str("module", "telegram").Err(err).Msg("reply aborted, retry limit exceeded")
			return nil
		}

		backoff := time.Second * 5 * time.Duration(try)
		log.Warn().Str("module", "telegram").Err(err).Str("sleep", backoff.String()).Msg("reply failed, sleeping and retrying")
		time.Sleep(backoff)
		try++
	}
}
