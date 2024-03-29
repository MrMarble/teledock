package telegram

import (
	"fmt"
	"math"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"

	"github.com/docker/docker/api/types"
	"github.com/mrmarble/teledock/internal/docker"
	"github.com/rs/zerolog"
	zero "github.com/rs/zerolog/log"
)

var log zerolog.Logger

// Telegram represents the telegram bot.
type Telegram struct {
	bot                *tb.Bot
	dckr               docker.Docker
	handlersRegistered bool
	admins             []int64
}

// Command represent a telegram command.
type Command struct {
	Cmd         string
	Aliases     []string
	Description string
	Handler     interface{}
}

// NewBot returns a Telegram bot.
func NewBot(token string, dckr docker.Docker, admins []int64) (*Telegram, error) {
	log = zero.With().Str("package", "Telegram").Logger()

	bot, err := tb.NewBot(tb.Settings{
		Token:  token,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		Reporter: func(err error) {
			log.Error().Err(err).Msg("telebot internal error")
		},
	})

	if err != nil {
		return nil, err
	}

	log.Info().Int64("id", bot.Me.ID).Str("name", bot.Me.FirstName).Str("username", bot.Me.Username).Msg("connected to telegram")

	return &Telegram{bot: bot, dckr: dckr, admins: admins}, nil
}

// Start starts polling for telegram updates.
func (t *Telegram) Start() {
	t.registerHandlers()

	log.Info().Msg("start polling")
	t.bot.Start()
}

// RegisterHandlers registers all the handlers.
func (t *Telegram) registerHandlers() {
	if t.handlersRegistered {
		return
	}

	log.Info().Msg("registering handlers")

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
		log.Fatal().Err(err).Msg("error registering commands")
	}

	t.handlersRegistered = true
}

func (t *Telegram) isSuperAdmin(user *tb.User) bool {
	for _, uid := range t.admins {
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
			log.Error().Err(err).Msg("send aborted, retry limit exceeded")
			return nil
		}

		backoff := time.Second * 5 * time.Duration(try)
		log.Warn().Err(err).Str("sleep", backoff.String()).Msg("send failed, sleeping and retrying")
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
			log.Error().Err(err).Msg("reply aborted, retry limit exceeded")
			return nil
		}

		backoff := time.Second * 5 * time.Duration(try)
		log.Warn().Err(err).Str("sleep", backoff.String()).Msg("reply failed, sleeping and retrying")
		time.Sleep(backoff)
		try++
	}
}

func (t *Telegram) makeContainerMenu(options types.ContainerListOptions, callback string) *tb.ReplyMarkup {
	buttonsPerRow := 3
	containers := t.dckr.List(options)

	menu := t.bot.NewMarkup()
	rowNumber := int(math.Ceil(float64(len(containers)) / float64(buttonsPerRow)))
	buttons := []tb.InlineButton{}
	rows := make([][]tb.InlineButton, rowNumber)
	for index, container := range containers {
		if index != 0 && index%buttonsPerRow == 0 {
			rows = append(rows, buttons)
			buttons = nil
		}

		btn := menu.Data(container.Names[0][1:], fmt.Sprintf("%v:%v", index, container.ID[:10]), fmt.Sprintf("%v:%v", callback, container.ID[:10])).Inline()
		buttons = append(buttons, *btn)
	}
	if len(buttons) > 0 {
		rows = append(rows, buttons)
	}
	menu.InlineKeyboard = rows
	return menu
}

func (t *Telegram) askForContainer(m *tb.Message, listOps types.ContainerListOptions, cb string) {
	t.reply(m, "Choose a container", t.makeContainerMenu(listOps, cb))
}

func (t *Telegram) handleLog(c *tb.Callback, payload string) {
	logs, err := t.dckr.Logs(payload, "10")
	if err != nil {
		t.callbackResponse(c, err, payload, "")
		return
	}
	for index, chunk := range logs {
		if index == 0 {
			t.callbackResponse(c, err, payload, fmt.Sprintf(FormatedStr, chunk))
		}
		if index != 0 && chunk != "" {
			t.send(c.Message.Chat, fmt.Sprintf(FormatedStr, chunk), tb.ModeHTML)
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (t *Telegram) callbackResponse(c *tb.Callback, err error, payload interface{}, response string) {
	if err != nil {
		err = t.bot.Respond(c, &tb.CallbackResponse{Text: err.Error(), ShowAlert: false})
		if err != nil {
			log.Fatal().Str("module", "utils").Err(err).Msg("error replying to message")
		}
		_, err = t.bot.Edit(c.Message, fmt.Sprintf("Container %v errored: %v", payload, err.Error()))
		if err != nil {
			log.Fatal().Str("module", "utils").Err(err).Msg("error editing message")
		}
	} else {
		err := t.bot.Respond(c, &tb.CallbackResponse{Text: "", ShowAlert: false})
		if err != nil {
			log.Fatal().Str("module", "utils").Err(err).Msg("error replying to message")
		}
		_, err = t.bot.Edit(c.Message, response, tb.ModeHTML)
		if err != nil {
			log.Fatal().Str("module", "utils").Err(err).Msg("error editing message")
		}
	}
}
