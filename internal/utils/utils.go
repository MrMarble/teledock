package utils

import (
	"encoding/json"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/enescakir/emoji"
	"github.com/mrmarble/teledock/internal/constants"
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

// parseInt64 parses a string and converts it to int64.
func ParseInt64(s string) (int64, error) {
	i, err := strconv.ParseInt(s, 10, 64)

	if err != nil {
		return 0, err
	}

	return i, nil
}

func IsNumber(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func FormatContainerList(containers []types.Container) []string {
	if len(containers) == 0 {
		return []string{"No containers running"}
	}
	resultMsg := make([]string, len(containers))
	for _, container := range containers {
		message := []string{
			fmt.Sprintf("%v  <b>%v</b>", state[container.State], container.Names[0][1:]),
			fmt.Sprintf(constants.FormatedStrPadded, "ID:", container.ID[:12]),
			fmt.Sprintf(constants.FormatedStrPadded, "STATUS:", container.Status),
			fmt.Sprintf(constants.FormatedStrPadded, "IMAGE:", container.Image),
		}
		if stack, ok := container.Labels[constants.ComposeLabel]; ok {
			message = append(message, fmt.Sprintf(constants.FormatedStrPadded, "STACK:", stack))
		}
		resultMsg = append(resultMsg, strings.Join(message, "\n"))
	}
	return resultMsg
}

func FormatImageList(images []types.ImageSummary) []string {
	if len(images) == 0 {
		return []string{"No images found"}
	}
	resultMsg := make([]string, len(images))
	for _, image := range images {
		resultMsg = append(resultMsg, fmt.Sprintf("<b>Tag: </b><code>%v</code>\n<b>ID: </b><code>%v</code>", html.EscapeString(image.RepoTags[0]), image.ID[6:18]))
	}
	return resultMsg
}

func FormatComposeList(compose map[string][]types.Container) []string {
	if len(compose) == 0 {
		return []string{"No stacks running"}
	}
	resultMsg := make([]string, len(compose))
	for stackName, stack := range compose {
		resultMsg = append(resultMsg, strings.Join([]string{
			fmt.Sprintf("<b>%v</b>", stackName),
			fmt.Sprintf(constants.FormatedStrPadded, "SERVICES:", len(stack)),
		}, "\n"))
	}
	return resultMsg
}

func FormatStruct(data interface{}) (string, error) {
	result, err := json.MarshalIndent(data, "", " ")
	if err != nil {
		return "", err
	}
	return string(result), nil
}

func ChunkString(s string, chunkSize int) []string {
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
