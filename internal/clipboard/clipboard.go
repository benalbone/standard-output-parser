package clipboard

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type command struct {
	name string
	args []string
}

// Copy writes text to the operating system clipboard using an available native tool.
func Copy(text string) error {
	commands := platformCommands()
	if len(commands) == 0 {
		return fmt.Errorf("clipboard copying is unsupported on %s", runtime.GOOS)
	}

	var lastErr error
	for _, candidate := range commands {
		path, err := exec.LookPath(candidate.name)
		if err != nil {
			lastErr = err
			continue
		}

		cmd := exec.Command(path, candidate.args...)
		cmd.Stdin = strings.NewReader(text)
		if err := cmd.Run(); err != nil {
			lastErr = err
			continue
		}
		return nil
	}

	if lastErr == nil {
		return fmt.Errorf("no clipboard command is available")
	}
	return fmt.Errorf("could not copy to clipboard: %w", lastErr)
}

func platformCommands() []command {
	switch runtime.GOOS {
	case "darwin":
		return []command{{name: "pbcopy"}}
	case "windows":
		return []command{{name: "cmd.exe", args: []string{"/c", "clip"}}}
	case "linux":
		return []command{
			{name: "wl-copy"},
			{name: "xclip", args: []string{"-selection", "clipboard"}},
			{name: "xsel", args: []string{"--clipboard", "--input"}},
		}
	default:
		return nil
	}
}
