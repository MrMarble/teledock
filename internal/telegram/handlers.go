package telegram

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/mrmarble/teledock/internal/utils"

	tb "gopkg.in/tucnak/telebot.v2"
)

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
	resultMsg := utils.FormatContainerList(t.dckr.List(types.ContainerListOptions{}))
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
}

// handleList triggers when the psa command is sent.
func (t *Telegram) handleListAll(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	resultMsg := utils.FormatContainerList(t.dckr.List(types.ContainerListOptions{All: true}))
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
}

func (t *Telegram) handleImageList(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	resultMsg := utils.FormatImageList(t.dckr.ListImages(types.ImageListOptions{}))
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
}

func (t *Telegram) handleStop(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}

	containerID := m.Payload
	if containerID == "" || !t.dckr.IsValidID(containerID) {
		t.askForContainer(m, types.ContainerListOptions{}, "stop")
	} else {
		if err := t.dckr.Stop(containerID); err != nil {
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
	if containerID == "" || !t.dckr.IsValidID(containerID) {
		filters := filters.NewArgs()
		filters.Add("status", "exited")
		t.askForContainer(m, types.ContainerListOptions{All: true, Filters: filters}, "start")
	} else {
		if err := t.dckr.Start(containerID); err != nil {
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
	if containerID == "" || !t.dckr.IsValidID(containerID) {
		t.askForContainer(m, types.ContainerListOptions{All: true}, "inspect")
		return
	}
	container, err := t.dckr.Inspect(containerID)
	if err != nil {
		t.reply(m, err.Error())
		return
	}

	response, err := utils.FormatStruct(container)
	if err != nil {
		t.reply(m, err.Error())
	}
	for index, chunk := range utils.ChunkString(response, 3000) {
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
	resultMsg := utils.FormatComposeList(t.dckr.ListCompose())
	t.send(m.Chat, strings.Join(resultMsg, "\n\n"))
}

func (t *Telegram) handleLogs(m *tb.Message) {
	if !t.isSuperAdmin(m.Sender) {
		return
	}
	payload := strings.Split(m.Payload, " ")
	containerID := payload[0]
	if containerID == "" || !t.dckr.IsValidID(containerID) {
		t.askForContainer(m, types.ContainerListOptions{All: true}, "logs")
	} else {
		tail := "10"
		if len(payload) > 1 {
			tail = payload[1]
		}
		logs, err := t.dckr.Logs(containerID, tail)

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

func (t *Telegram) inspectHandler(c *tb.Callback, payload string) {
	container, err := t.dckr.Inspect(payload)
	if err != nil {
		t.callbackResponse(c, err, payload, "")
		return
	}

	response, err := utils.FormatStruct(container)
	if err != nil {
		t.callbackResponse(c, err, payload, "")
		return
	}
	for index, chunk := range utils.ChunkString(response, 3000) {
		if index == 0 {
			t.callbackResponse(c, err, payload, fmt.Sprintf(FormatedStr, chunk))
		} else {
			t.send(c.Message.Chat, fmt.Sprintf(FormatedStr, chunk), tb.ModeHTML)
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func (t *Telegram) handleCallback(c *tb.Callback) {
	parts := strings.Split(c.Data, ":")
	instruction := parts[0]
	payload := parts[1]

	switch instruction {
	case "stop":
		err := t.dckr.Stop(payload)
		t.callbackResponse(c, err, payload, fmt.Sprintf("Container %v stopped", payload))

	case "start":
		err := t.dckr.Start(payload)
		t.callbackResponse(c, err, payload, fmt.Sprintf("Container %v started", payload))

	case "inspect":
		t.inspectHandler(c, payload)

	case "logs":
		t.handleLog(c, payload)
	}
}
