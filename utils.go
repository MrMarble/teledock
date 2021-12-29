package main

import (
	"encoding/json"
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/rs/zerolog/log"
	tb "gopkg.in/tucnak/telebot.v2"
)

const ComposeLabel = "com.docker.compose.project"
const FormatedStrPadded = "<code> %-8v</code><code>%v</code>"

// parseInt64 parses a string and converts it to int64.
func parseInt64(s string) (int64, error) {
	i, err := strconv.ParseInt(s, 10, 64)

	if err != nil {
		return 0, err
	}

	return i, nil
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
			fmt.Sprintf(FormatedStrPadded, "ID:", container.ID[:12]),
			fmt.Sprintf(FormatedStrPadded, "STATUS:", container.Status),
			fmt.Sprintf(FormatedStrPadded, "IMAGE:", container.Image),
		}
		if stack, ok := container.Labels[ComposeLabel]; ok {
			message = append(message, fmt.Sprintf(FormatedStrPadded, "STACK:", stack))
		}
		resultMsg = append(resultMsg, strings.Join(message, "\n"))
	}
	return resultMsg
}

func parseImageList(options types.ImageListOptions) []string {
	images := docker.listImages(options)
	if len(images) == 0 {
		return []string{"No images found"}
	}
	resultMsg := make([]string, len(images))
	for _, image := range images {
		resultMsg = append(resultMsg, fmt.Sprintf("<b>Tag: </b><code>%v</code>\n<b>ID: </b><code>%v</code>", html.EscapeString(image.RepoTags[0]), image.ID[6:18]))
	}
	return resultMsg
}

func getStacks() map[string][]types.Container {
	var (
		filters = filters.NewArgs()
		stacks  = map[string][]types.Container{}
	)
	filters.Add("label", ComposeLabel)
	containers := docker.list(types.ContainerListOptions{All: true, Filters: filters})
	for _, container := range containers {
		stacks[container.Labels[ComposeLabel]] = append(stacks[container.Labels[ComposeLabel]], container)
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
			fmt.Sprintf(FormatedStrPadded, "SERVICES:", len(stack)),
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

func handleInspect(t *Telegram, c *tb.Callback, payload string) {
	container, err := docker.inspect(payload)
	if err != nil {
		callbackResponse(t, c, err, payload, "")
		return
	}

	response, err := formatStruct(container)
	if err != nil {
		callbackResponse(t, c, err, payload, "")
		return
	}
	for index, chunk := range chunkString(response, 3000) {
		if index == 0 {
			callbackResponse(t, c, err, payload, fmt.Sprintf(FormatedStr, chunk))
		} else {
			t.send(c.Message.Chat, fmt.Sprintf(FormatedStr, chunk), tb.ModeHTML)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func handleLog(t *Telegram, c *tb.Callback, payload string) {
	logs, err := docker.logs(payload, "10")
	if err != nil {
		callbackResponse(t, c, err, payload, "")
		return
	}
	for index, chunk := range logs {
		if index == 0 {
			callbackResponse(t, c, err, payload, fmt.Sprintf(FormatedStr, chunk))
		}
		if index != 0 && chunk != "" {
			t.send(c.Message.Chat, fmt.Sprintf(FormatedStr, chunk), tb.ModeHTML)
		}
		time.Sleep(100 * time.Millisecond)
	}
}
