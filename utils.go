package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/rs/zerolog/log"
	tb "gopkg.in/tucnak/telebot.v2"
)

const COMPOSE_LABEL = "com.docker.compose.project"
const FORMATED_STR_PADDED = "<code> %-8v</code><code>%v</code>"

// parseInt64 parses a string and converts it to int64
func parseInt64(s string) (int64, error) {
	i, err := strconv.ParseInt(s, 10, 32)

	if err != nil {
		return 0, err
	}

	return int64(i), nil
}

func isNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func parseList(options types.ContainerListOptions) []string {
	containers := docker.list(options)
	if len(containers) == 0 {
		return []string{"No containers running"}
	}
	resultMsg := make([]string, len(containers))
	for _, container := range containers {
		message := []string{
			fmt.Sprintf("%v  <b>%v</b>", state[container.State], container.Names[0][1:]),
			fmt.Sprintf(FORMATED_STR_PADDED, "ID:", container.ID[:12]),
			fmt.Sprintf(FORMATED_STR_PADDED, "STATUS:", container.Status),
			fmt.Sprintf(FORMATED_STR_PADDED, "IMAGE:", container.Image),
		}
		if stack, ok := container.Labels[COMPOSE_LABEL]; ok {
			message = append(message, fmt.Sprintf(FORMATED_STR_PADDED, "STACK:", stack))
		}
		resultMsg = append(resultMsg, strings.Join(message, "\n"))
	}
	return resultMsg
}

func getStacks() map[string][]types.Container {
	var (
		filters = filters.NewArgs()
		stacks  = map[string][]types.Container{}
	)
	filters.Add("label", COMPOSE_LABEL)
	containers := docker.list(types.ContainerListOptions{All: true, Filters: filters})
	for _, container := range containers {
		stacks[container.Labels[COMPOSE_LABEL]] = append(stacks[container.Labels[COMPOSE_LABEL]], container)
	}
	return stacks
}

func parseStacks() []string {
	stacks := getStacks()
	if len(stacks) == 0 {
		return []string{"No stacks running"}
	}
	resultMsg := make([]string, len(stacks))
	for stackName, stack := range stacks {
		resultMsg = append(resultMsg, strings.Join([]string{
			fmt.Sprintf("<b>%v</b>", stackName),
			fmt.Sprintf(FORMATED_STR_PADDED, "SERVICES:", len(stack)),
		}, "\n"))
	}
	return resultMsg
}

func makeContainerMenu(t *Telegram, options types.ContainerListOptions, callback string) *tb.ReplyMarkup {
	buttonsPerRow := 3
	containers := docker.list(options)

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

func formatStruct(data interface{}) (string, error) {
	result, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func callbackResponse(t *Telegram, c *tb.Callback, err error, payload interface{}, response string) {
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

func chunkString(s string, chunkSize int) []string {
	var chunks []string
	runes := []rune(s)

	if len(runes) == 0 {
		return []string{s}
	}

	for i := 0; i < len(runes); i += chunkSize {
		nn := i + chunkSize
		if nn > len(runes) {
			nn = len(runes)
		}
		chunks = append(chunks, string(runes[i:nn]))
	}
	return chunks
}

func (t *Telegram) askForContainer(m *tb.Message, listOps types.ContainerListOptions, cb string) {
	t.reply(m, "Choose a container", makeContainerMenu(t, listOps, cb))
}
