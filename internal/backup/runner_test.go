package backup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

func TestHelperProcess(t *testing.T) {
	mode := ""
	for i, a := range os.Args {
		if a == "--" && i+1 < len(os.Args) {
			mode = os.Args[i+1]
		}
	}
	if mode == "" {
		return
	}
	switch mode {
	case "envdump":
		for _, e := range os.Environ() {
			fmt.Println(e)
		}
	case "bigout":
		_, _ = os.Stdout.Write(make([]byte, 8192))
	case "sleep":
		time.Sleep(60 * time.Second)
	case "secretstderr":
		fmt.Fprint(os.Stderr, "STDERR_SECRET_SENTINEL")
		os.Exit(1)
	case "version":
		fmt.Println("fnox 1.35.2")
	case "badversion":
		fmt.Println("fnox 9.9.9")
	}
	os.Exit(0)
}

func helperArgs(mode string) []string {
	return []string{"-test.run=TestHelperProcess", "--", mode}
}

func TestExecRunnerEnvIsolation(t *testing.T) {
	t.Setenv("FNOX_AGE_KEY", "sentinel")
	t.Setenv("RUST_LOG", "debug")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "sentinel")
	dir := t.TempDir()
	env, err := isolatedEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	out, err := execRunner{}.Run(context.Background(), os.Args[0], helperArgs("envdump"), env, dir, nil, 65536)
	if err != nil {
		t.Fatalf("helper failed: %v", err)
	}
	if strings.Contains(string(out), "sentinel") || strings.Contains(string(out), "RUST_LOG") {
		t.Fatalf("sentinel env leaked to child: %s", out)
	}
	found := map[string]bool{}
	for _, line := range strings.Split(string(out), "\n") {
		if k := strings.SplitN(line, "=", 2)[0]; k != "" {
			found[k] = true
		}
	}
	for _, want := range []string{"FNOX_CONFIG_DIR", "NO_COLOR", "PATH"} {
		if !found[want] {
			t.Fatalf("expected %s in child env", want)
		}
	}
}

func TestExecRunnerOutputBound(t *testing.T) {
	dir := t.TempDir()
	env, err := isolatedEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = execRunner{}.Run(context.Background(), os.Args[0], helperArgs("bigout"), env, dir, nil, 1024)
	if err == nil {
		t.Fatal("expected output limit error")
	}
	if strings.Contains(err.Error(), "bigout") {
		t.Fatal("error leaked output")
	}
}

func TestExecRunnerTimeoutAndCancel(t *testing.T) {
	dir := t.TempDir()
	env, err := isolatedEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_, err = execRunner{}.Run(ctx, os.Args[0], helperArgs("sleep"), env, dir, nil, 1024)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}

	ctx2, cancel2 := context.WithCancel(context.Background())
	cancel2()
	_, err = execRunner{}.Run(ctx2, os.Args[0], helperArgs("envdump"), env, dir, nil, 1024)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected Canceled, got %v", err)
	}
}

func TestExecRunnerStderrNotLeaked(t *testing.T) {
	dir := t.TempDir()
	env, err := isolatedEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	_, err = execRunner{}.Run(context.Background(), os.Args[0], helperArgs("secretstderr"), env, dir, nil, 1024)
	if err == nil {
		t.Fatal("expected error")
	}
	if strings.Contains(err.Error(), "STDERR_SECRET_SENTINEL") {
		t.Fatal("stderr leaked into error")
	}
}

func TestCheckFnoxVersion(t *testing.T) {
	dir := t.TempDir()
	env, err := isolatedEnv(dir)
	if err != nil {
		t.Fatal(err)
	}
	runner := &fakeVersionRunner{out: "fnox 1.35.2\n"}
	if err := checkFnoxVersion(context.Background(), runner, "fnox", env, dir); err != nil {
		t.Fatalf("expected version accepted, got %v", err)
	}
	runner.out = "fnox 9.9.9\n"
	if err := checkFnoxVersion(context.Background(), runner, "fnox", env, dir); err == nil {
		t.Fatal("expected version rejection")
	}
}

type fakeVersionRunner struct {
	out string
	err error
}

func (f *fakeVersionRunner) Run(ctx context.Context, binary string, args []string, env []string, cwd string, stdin []byte, maxStdout int) ([]byte, error) {
	return []byte(f.out), f.err
}
