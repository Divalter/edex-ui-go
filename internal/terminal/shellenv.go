package terminal

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const envDelimiter = "_SHELL_ENV_DELIMITER_"

// ShellEnv returns the environment of a login shell, like the shell-env
// module used by the original (see eDEX-UI #366): apps started from a desktop
// launcher often miss the variables set in the user's shell profile.
func ShellEnv(shell string) []string {
	if runtime.GOOS == "windows" {
		return os.Environ()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	script := "echo -n " + envDelimiter + "; env; echo -n " + envDelimiter + "; exit"
	cmd := exec.CommandContext(ctx, shell, "-ilc", script)
	cmd.Env = append(os.Environ(), "DISABLE_AUTO_UPDATE=true")
	out, err := cmd.Output()
	if err != nil {
		return os.Environ()
	}
	parts := bytes.Split(out, []byte(envDelimiter))
	if len(parts) < 3 {
		return os.Environ()
	}
	var env []string
	for _, line := range strings.Split(string(parts[1]), "\n") {
		if i := strings.IndexByte(line, '='); i > 0 {
			env = append(env, line)
		}
	}
	if len(env) == 0 {
		return os.Environ()
	}
	return env
}

// MergeEnv sets the given variables in env, replacing existing ones.
func MergeEnv(env []string, vars map[string]string) []string {
	out := make([]string, 0, len(env)+len(vars))
	for _, kv := range env {
		key := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			key = kv[:i]
		}
		if _, replaced := vars[key]; !replaced {
			out = append(out, kv)
		}
	}
	for k, v := range vars {
		out = append(out, k+"="+v)
	}
	return out
}
