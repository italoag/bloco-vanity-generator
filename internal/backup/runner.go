package backup

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type commandRunner interface {
	Run(ctx context.Context, binary string, args []string, env []string, cwd string, stdin []byte, maxStdout int) ([]byte, error)
}

type execRunner struct{}

type limitWriter struct {
	buf      bytes.Buffer
	max      int
	exceeded bool
}

func (w *limitWriter) Write(p []byte) (int, error) {
	if w.exceeded {
		return 0, fmt.Errorf("output limit exceeded")
	}
	remaining := w.max - w.buf.Len()
	if len(p) > remaining {
		if remaining > 0 {
			w.buf.Write(p[:remaining])
		}
		w.exceeded = true
		return 0, fmt.Errorf("output limit exceeded")
	}
	return w.buf.Write(p)
}

func (execRunner) Run(ctx context.Context, binary string, args []string, env []string, cwd string, stdin []byte, maxStdout int) ([]byte, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Dir = cwd
	cmd.Env = env
	cmd.WaitDelay = time.Second
	cmd.Stdin = bytes.NewReader(stdin)
	stdout := &limitWriter{max: maxStdout}
	cmd.Stdout = stdout
	cmd.Stderr = &limitWriter{max: 4096}

	runErr := cmd.Run()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if stdout.exceeded {
		return nil, fmt.Errorf("fnox output exceeded limit")
	}
	if runErr != nil {
		return nil, fmt.Errorf("fnox command failed")
	}
	return stdout.buf.Bytes(), nil
}

var envAllowlist = []string{
	"PATH", "HOME", "USER", "LOGNAME", "USERPROFILE",
	"SYSTEMROOT", "TMPDIR", "TEMP", "TMP", "LANG",
}

func isolatedEnv(scratch string) ([]string, error) {
	isolated := filepath.Join(scratch, "isolated")
	if err := os.MkdirAll(isolated, 0700); err != nil {
		return nil, err
	}
	env := make([]string, 0, len(envAllowlist)+2)
	for _, key := range envAllowlist {
		if value, ok := os.LookupEnv(key); ok {
			env = append(env, key+"="+value)
		}
	}
	env = append(env, "FNOX_CONFIG_DIR="+isolated, "NO_COLOR=1")
	return env, nil
}

func checkFnoxVersion(ctx context.Context, runner commandRunner, binary string, env []string, cwd string) error {
	out, err := runner.Run(ctx, binary, []string{"--version"}, env, cwd, nil, 4096)
	if err != nil {
		return safeCommandError(err, "fnox version check failed")
	}
	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) != 2 || fields[0] != "fnox" || fields[1] != SupportedFnoxVersion {
		return fmt.Errorf("unsupported fnox version")
	}
	return nil
}
