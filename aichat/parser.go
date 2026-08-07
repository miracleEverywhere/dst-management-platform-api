package aichat

import (
	"regexp"
	"strings"
)

var sayLogPattern = regexp.MustCompile(`^\[[^\]]+\]:\s*\[Say\]\s*\(([^)]+)\)\s+(.+?):\s*(.*)$`)

type chatEvent struct {
	UID      string
	Nickname string
	Message  string
}

func parseChatEvent(line string) (chatEvent, bool) {
	matches := sayLogPattern.FindStringSubmatch(strings.TrimSpace(line))
	if len(matches) != 4 {
		return chatEvent{}, false
	}

	event := chatEvent{
		UID:      strings.TrimSpace(matches[1]),
		Nickname: strings.TrimSpace(matches[2]),
		Message:  strings.TrimSpace(matches[3]),
	}
	if event.UID == "" || event.Nickname == "" || event.Message == "" {
		return chatEvent{}, false
	}
	return event, true
}
