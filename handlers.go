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

// handleStart triggers when /start is sent on private
func (t *Telegram) handleStart(m *tb.Message) {
	if !m.Private() {
		return
	}

	t.send(m.Chat, "Telegram bot made by <a href='tg://user?id=256671105'>MrMarble</a>")
}

// handleList triggers when the ps command is sent
func (t *Telegram) handleList(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	resultMsg := parseList(types.ContainerListOptions{})
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
}

// handleList triggers when the psa command is sent
func (t *Telegram) handleListAll(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	resultMsg := parseList(types.ContainerListOptions{All: true})
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
}

func (t *Telegram) handleStop(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}

	containerID := m.Payload
	if containerID == "" || !docker.isValidID(containerID) {
		t.reply(m, "Choose a container", makeContainerMenu(t, types.ContainerListOptions{}, "stop"))
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
		t.reply(m, "Choose a container", makeContainerMenu(t, types.ContainerListOptions{All: true, Filters: filters}, "start"))
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
		t.reply(m, "Choose a container", makeContainerMenu(t, types.ContainerListOptions{All: true}, "inspect"))
	} else {
		container, err := docker.inspect(containerID)
		if err != nil {
			t.reply(m, err.Error())
		} else {
			response, err := formatStruct(container)
			if err != nil {
				t.reply(m, err.Error())
			}
			for index, chunk := range chunkString(response, 399) {
				if index == 0 {
					t.reply(m, fmt.Sprintf("<code>%v</code>", chunk), tb.ModeHTML)
				} else {
					t.send(m.Chat, fmt.Sprintf("<code>%v</code>", chunk), tb.ModeHTML)
				}
				time.Sleep(100 * time.Millisecond)
			}
		}
	}
}

func (t *Telegram) handleStacks(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	resultMsg := parseStacks()
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
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
		container, err := docker.inspect(payload)
		if err != nil {
			callbackResponse(t, c, err, payload, fmt.Sprintf("<code>%v</code>", payload))
		} else {
			response, err := formatStruct(container)
			if err != nil {
				callbackResponse(t, c, err, payload, fmt.Sprintf("<code>%v</code>", response))
				return
			}
			for index, chunk := range chunkString(response, 300) {
				if index == 0 {
					callbackResponse(t, c, err, payload, fmt.Sprintf("<code>%v</code>", chunk))
				} else {
					t.send(c.Message.Chat, fmt.Sprintf("<code>%v</code>", chunk), tb.ModeHTML)
				}
				time.Sleep(250 * time.Millisecond)
			}
		}

	}
}
