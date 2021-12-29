package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/enescakir/emoji"

	tb "gopkg.in/tucnak/telebot.v2"
)

var state = map[string]emoji.Emoji{
	"running":    emoji.CheckMarkButton,
	"created":    emoji.Egg,
	"restarting": emoji.RecyclingSymbol,
	"removing":   emoji.Wastebasket,
	"paused":     emoji.PauseButton,
	"exited":     emoji.NoEntry,
	"dead":       emoji.Skull,
}

const FormatedStr = "<code>%v</code>"

// handleStart triggers when /start is sent on private.
func (t *Telegram) handleStart(m *tb.Message) {
	if !m.Private() {
		return
	}

	t.send(m.Chat, "Telegram bot made by <a href='tg://user?id=256671105'>MrMarble</a>")
}

// handleList triggers when the ps command is sent.
func (t *Telegram) handleList(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	resultMsg := parseList(types.ContainerListOptions{})
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
}

// handleList triggers when the psa command is sent.
func (t *Telegram) handleListAll(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	resultMsg := parseList(types.ContainerListOptions{All: true})
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
}

func (t *Telegram) handleImageList(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	resultMsg := parseImageList(types.ImageListOptions{})
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
}

func (t *Telegram) handleStop(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}

	containerID := m.Payload
	if containerID == "" || !docker.isValidID(containerID) {
		t.askForContainer(m, types.ContainerListOptions{}, "stop")
	} else {
		if err := docker.stop(containerID); err != nil {
			t.reply(m, err.Error())
		} else {
			t.reply(m, "Container stopped")
		}
	}
}

func (t *Telegram) handleStartContainer(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}

	containerID := m.Payload
	if containerID == "" || !docker.isValidID(containerID) {
		filters := filters.NewArgs()
		filters.Add("status", "exited")
		t.askForContainer(m, types.ContainerListOptions{All: true, Filters: filters}, "start")
	} else {
		if err := docker.start(containerID); err != nil {
			t.reply(m, err.Error())
		} else {
			t.reply(m, "Container started")
		}
	}
}

func (t *Telegram) handleInspect(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}

	containerID := m.Payload
	if containerID == "" || !docker.isValidID(containerID) {
		t.askForContainer(m, types.ContainerListOptions{All: true}, "inspect")
		return
	}
	container, err := docker.inspect(containerID)
	if err != nil {
		t.reply(m, err.Error())
		return
	}

	response, err := formatStruct(container)
	if err != nil {
		t.reply(m, err.Error())
	}
	for index, chunk := range chunkString(response, 3000) {
		if index == 0 {
			t.reply(m, fmt.Sprintf(FormatedStr, chunk), tb.ModeHTML)
		} else {
			t.send(m.Chat, fmt.Sprintf(FormatedStr, chunk), tb.ModeHTML)
		}
		time.Sleep(100 * time.Millisecond)
	}

}

func (t *Telegram) handleStacks(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	resultMsg := parseStacks()
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
}

func (t *Telegram) handleLogs(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	payload := strings.Split(m.Payload, " ")
	containerID := payload[0]
	if containerID == "" || !docker.isValidID(containerID) {
		t.askForContainer(m, types.ContainerListOptions{All: true}, "logs")
	} else {
		tail := "10"
		if len(payload) > 1 {
			tail = payload[1]
		}
		logs, err := docker.logs(containerID, tail)

		if err != nil {
			t.reply(m, err.Error())
			return
		}
		for index, chunk := range logs {
			if index == 0 {
				t.reply(m, fmt.Sprintf(FormatedStr, chunk), tb.ModeHTML)
			} else {
				t.send(m.Chat, fmt.Sprintf(FormatedStr, chunk), tb.ModeHTML)
			}
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (t *Telegram) handleCallback(c *tb.Callback) {
	parts := strings.Split(c.Data, ":")
	instruction := parts[0]
	payload := parts[1]

	switch instruction {
	case "stop":
		err := docker.stop(payload)
		callbackResponse(t, c, err, payload, fmt.Sprintf("Container %v stopped", payload))

	case "start":
		err := docker.start(payload)
		callbackResponse(t, c, err, payload, fmt.Sprintf("Container %v started", payload))

	case "inspect":
		handleInspect(t, c, payload)

	case "logs":
		handleLog(t, c, payload)
	}
}
