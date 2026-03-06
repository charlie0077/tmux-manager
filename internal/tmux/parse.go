package tmux

import (
	"fmt"
	"strings"
	"time"
)

type Session struct {
	Name     string
	Windows  int
	Created  time.Time
	Attached bool
}

// ParseTmuxLS parses the output of `tmux ls -F '#{session_name}|#{session_windows}|#{session_created}|#{session_attached}'`
func ParseTmuxLS(output string) []Session {
	var sessions []Session
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 4 {
			continue
		}

		windows := atoi(parts[1])

		var created time.Time
		if ts, err := parseUnixTimestamp(parts[2]); err == nil {
			created = ts
		}

		attached := parts[3] == "1"

		sessions = append(sessions, Session{
			Name:     parts[0],
			Windows:  windows,
			Created:  created,
			Attached: attached,
		})
	}
	return sessions
}

func atoi(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}

func parseUnixTimestamp(s string) (time.Time, error) {
	var ts int64
	for _, c := range s {
		if c >= '0' && c <= '9' {
			ts = ts*10 + int64(c-'0')
		} else {
			return time.Time{}, fmt.Errorf("invalid timestamp: %s", s)
		}
	}
	return time.Unix(ts, 0), nil
}
