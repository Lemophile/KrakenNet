package pkg

import (
	"bufio"
	"fmt"
	color "github.com/vexsx/KrakenNet/config"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

type Inputs struct {
	Target      string
	Method      int
	Connections int
	Workers     int
	Duration    time.Duration
	Port        int
}

// You provide the validators so PromptInputs stays generic.
func PromptInputs() (*Inputs, error) {
	r := bufio.NewReader(os.Stdin)
	w := os.Stdout

	target, err := promptNonEmptyString(r, w, "🌐 Target (URL or IP): ")
	if err != nil {
		return nil, err
	}

	method, err := promptIntWithValidator(
		r, w,
		"🛠 Methods :\n1-Kraken\n2-TLS\n3-UDP Discord\n4-UDP Bypass\n5-UDP gbps\nJust enter method number: ",
		isValidMethod,
		"Input was incorrect, try again.",
	)
	if err != nil {
		return nil, err
	}

	var port int
	if requiresPort(method) {
		// generic target port prompt; default 5000 if blank/invalid/<1
		port, err = promptPortWithDefault(r, w, "🎯 Port (target): ", 5000, 1)
		if err != nil {
			return nil, err
		}
	}

	connections, err := promptIntWithDefault(r, w, "🔗 Connections per worker: ", 10, 1)
	if err != nil {
		return nil, err
	}

	workers, err := promptIntWithDefault(r, w, "🔧 Number of workers: ", 10, 1)
	if err != nil {
		return nil, err
	}

	seconds, err := promptIntWithDefault(r, w, "⏱️ Duration (in seconds): ", 30, 1)
	if err != nil {
		return nil, err
	}

	return &Inputs{
		Target:      target,
		Method:      method,
		Connections: connections,
		Workers:     workers,
		Duration:    time.Duration(seconds) * time.Second,
		Port:        port,
	}, nil
}

// ---- Helpers (colored + error-aware prints) ----
func cprint(w io.Writer, s string) error {
	if _, err := fmt.Fprint(w, color.Yellow); err != nil {
		return err
	}
	if _, err := fmt.Fprint(w, s); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, color.Reset)
	return err
}
func cprintln(w io.Writer, s string) error {
	if err := cprint(w, s); err != nil {
		return err
	}
	_, err := fmt.Fprint(w, "\n")
	return err
}

func promptNonEmptyString(r *bufio.Reader, w io.Writer, prompt string) (string, error) {
	for {
		if err := cprint(w, prompt); err != nil {
			return "", err
		}
		s, err := r.ReadString('\n')
		if err == nil {
			s = strings.TrimSpace(s)
			if s != "" {
				return s, nil
			}
		}
		if err := cprintln(w, "Input was incorrect, try again."); err != nil {
			return "", err
		}
	}
}

func promptIntWithDefault(r *bufio.Reader, w io.Writer, prompt string, def, min int) (int, error) {
	for {
		if err := cprint(w, prompt); err != nil {
			return 0, err
		}
		s, err := r.ReadString('\n')
		if err != nil {
			if err := cprintln(w, "Failed to read input, try again."); err != nil {
				return 0, err
			}
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return def, nil
		}
		v, err := strconv.Atoi(s)
		if err != nil {
			if err := cprintln(w, "Invalid input, please enter a number."); err != nil {
				return 0, err
			}
			continue
		}
		if v < min {
			return def, nil
		}
		return v, nil
	}
}

func promptIntWithValidator(
	r *bufio.Reader,
	w io.Writer,
	prompt string,
	valid func(int) bool,
	errMsg string,
) (int, error) {
	for {
		if err := cprint(w, prompt); err != nil {
			return 0, err
		}
		s, err := r.ReadString('\n')
		if err != nil {
			if err := cprintln(w, errMsg); err != nil {
				return 0, err
			}
			continue
		}
		v, convErr := strconv.Atoi(strings.TrimSpace(s))
		if convErr != nil || !valid(v) {
			if err := cprintln(w, errMsg); err != nil {
				return 0, err
			}
			continue
		}
		return v, nil
	}
}

// specifically for a port with default + min guard
func promptPortWithDefault(r *bufio.Reader, w io.Writer, prompt string, def, min int) (int, error) {
	for {
		if err := cprint(w, prompt); err != nil {
			return 0, err
		}
		s, err := r.ReadString('\n')
		if err != nil {
			if err := cprintln(w, "Failed to read input, try again."); err != nil {
				return 0, err
			}
			continue
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return def, nil
		}
		v, err := strconv.Atoi(s)
		if err != nil {
			if err := cprintln(w, "Invalid input, please enter a number."); err != nil {
				return 0, err
			}
			continue
		}
		if v < min {
			return def, nil
		}
		return v, nil
	}
}
func isValidMethod(mode int) bool {
	switch mode {
	case 1, 2, 3, 4, 5:
		return true
	default:
		return false
	}
}
func requiresPort(mode int) bool {
	return mode == 3 || mode == 4 || mode == 5
}
