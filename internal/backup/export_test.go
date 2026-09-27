package backup

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExportEthereum(t *testing.T) {
	b := ethereumBundle(t, true)
	base := t.TempDir()
	outDir := filepath.Join(base, "export")
	if err := Export(context.Background(), b, outDir); err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	info, err := os.Lstat(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0700 {
		t.Fatalf("export dir mode %o", info.Mode().Perm())
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name()] = true
		info, err := e.Info()
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("%s mode %o", e.Name(), info.Mode().Perm())
		}
	}
	for _, want := range []string{"wallet-backup.json", b.Address + ".json", b.Address + ".pwd", b.Address + ".mnemonic"} {
		if !names[want] {
			t.Fatalf("missing %s; got %v", want, names)
		}
	}
	pwd, err := os.ReadFile(filepath.Join(outDir, b.Address+".pwd"))
	if err != nil {
		t.Fatal(err)
	}
	if string(pwd) != b.KeystorePassword {
		t.Fatal("password file mismatch")
	}
	mn, err := os.ReadFile(filepath.Join(outDir, b.Address+".mnemonic"))
	if err != nil {
		t.Fatal(err)
	}
	if string(mn) != b.Mnemonic {
		t.Fatal("mnemonic file mismatch")
	}
	var decoded Bundle
	data, err := os.ReadFile(filepath.Join(outDir, "wallet-backup.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.PrivateKey != b.PrivateKey {
		t.Fatal("bundle payload mismatch")
	}
}

func TestExportBitcoin(t *testing.T) {
	b := bitcoinBundle(t, true)
	outDir := filepath.Join(t.TempDir(), "export")
	if err := Export(context.Background(), b, outDir); err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, e := range entries {
		names[e.Name()] = true
	}
	if !names["wallet-backup.json"] || !names[b.Address+".key"] {
		t.Fatalf("missing artifacts: %v", names)
	}
	for name := range names {
		if filepath.Ext(name) == ".mnemonic" {
			t.Fatal("bitcoin export wrote unrelated mnemonic")
		}
	}
	key, err := os.ReadFile(filepath.Join(outDir, b.Address+".key"))
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != b.PrivateKey {
		t.Fatal("key file mismatch")
	}
}

func TestExportSolana(t *testing.T) {
	b := solanaBundle(t)
	outDir := filepath.Join(t.TempDir(), "export")
	if err := Export(context.Background(), b, outDir); err != nil {
		t.Fatalf("Export failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(outDir, b.Address+".json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || data[0] != '[' {
		t.Fatal("solana keypair must be a numeric JSON array")
	}
	var keyArr []int
	if err := json.Unmarshal(data, &keyArr); err != nil {
		t.Fatal(err)
	}
	if len(keyArr) != 64 {
		t.Fatalf("expected 64 elements, got %d", len(keyArr))
	}
	wantKey, err := hex.DecodeString(b.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}
	for i, v := range keyArr {
		if v < 0 || v > 255 || byte(v) != wantKey[i] {
			t.Fatalf("keypair element %d mismatch", i)
		}
	}
	key, err := os.ReadFile(filepath.Join(outDir, b.Address+".key"))
	if err != nil {
		t.Fatal(err)
	}
	if string(key) != b.PrivateKey {
		t.Fatal("key file mismatch")
	}
}

func TestExportExistingDestUntouched(t *testing.T) {
	b := ethereumBundle(t, false)
	outDir := filepath.Join(t.TempDir(), "export")
	if err := os.Mkdir(outDir, 0700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(outDir, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Export(context.Background(), b, outDir); err == nil {
		t.Fatal("expected error")
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "keep" {
		t.Fatal("preexisting directory modified")
	}
}

func TestExportCancelled(t *testing.T) {
	b := ethereumBundle(t, false)
	outDir := filepath.Join(t.TempDir(), "export")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Export(ctx, b, outDir); err == nil {
		t.Fatal("expected error")
	}
	if _, err := os.Lstat(outDir); !os.IsNotExist(err) {
		t.Fatal("export dir left behind on cancellation")
	}
}

type failAfterChecksContext struct {
	context.Context
	calls int
	fail  int
}

func (c *failAfterChecksContext) Err() error {
	c.calls++
	if c.calls > c.fail {
		return context.Canceled
	}
	return c.Context.Err()
}

func TestExportRollbackOnWriteFault(t *testing.T) {
	b := ethereumBundle(t, false)
	base := t.TempDir()
	outDir := filepath.Join(base, "export")
	existing := filepath.Join(base, "unrelated.txt")
	if err := os.WriteFile(existing, []byte("keep"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx := &failAfterChecksContext{Context: context.Background(), fail: 1}
	if err := Export(ctx, b, outDir); err == nil {
		t.Fatal("expected error")
	}
	if _, lerr := os.Lstat(outDir); !os.IsNotExist(lerr) {
		t.Fatal("export dir not rolled back")
	}
	data, err := os.ReadFile(existing)
	if err != nil || string(data) != "keep" {
		t.Fatal("unrelated file modified")
	}
}

func TestWriteExportFile(t *testing.T) {
	payload := []byte(`{"x":1}`)

	assertNoTemps := func(t *testing.T, dir string) {
		t.Helper()
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir: %v", err)
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), ".export-tmp-") {
				t.Fatal("stale temp left behind")
			}
		}
	}

	t.Run("success_stages_and_links", func(t *testing.T) {
		dir := t.TempDir()
		final := filepath.Join(dir, "wallet-backup.json")
		var publishBytes []byte
		var publishPerm os.FileMode
		err := writeExportFileWith(final, payload, writeSyncClose, func(tmp, dst string) error {
			if _, lerr := os.Lstat(dst); !os.IsNotExist(lerr) {
				t.Fatal("final must not exist before publish")
			}
			b, rerr := os.ReadFile(tmp)
			if rerr != nil {
				return rerr
			}
			publishBytes = b
			info, serr := os.Lstat(tmp)
			if serr != nil {
				return serr
			}
			publishPerm = info.Mode().Perm()
			return os.Link(tmp, dst)
		})
		if err != nil {
			t.Fatal(err)
		}
		if string(publishBytes) != string(payload) {
			t.Fatal("publish must see complete bytes")
		}
		if publishPerm != 0600 {
			t.Fatalf("staged perm %v", publishPerm)
		}
		got, err := os.ReadFile(final)
		if err != nil {
			t.Fatalf("read final: %v", err)
		}
		if string(got) != string(payload) {
			t.Fatal("final missing or wrong content")
		}
		assertNoTemps(t, dir)
		entries, err := os.ReadDir(dir)
		if err != nil {
			t.Fatalf("read dir: %v", err)
		}
		if len(entries) != 1 {
			t.Fatal("temp file not cleaned")
		}
	})

	t.Run("write_error_leaves_no_final_or_temp", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "partial.json")
		write := func(f *os.File, data []byte) error {
			if _, err := f.Write(data[:3]); err != nil {
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			return fmt.Errorf("injected write failure")
		}
		err := writeExportFileWith(target, payload, write, os.Link)
		if err == nil {
			t.Fatal("expected error")
		}
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Fatal("final must be absent")
		}
		assertNoTemps(t, dir)
	})

	t.Run("existing_file_preserved", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "existing.json")
		if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
			t.Fatal(err)
		}
		err := writeExportFileWith(target, payload, writeSyncClose, os.Link)
		if err == nil || !errors.Is(err, os.ErrExist) {
			t.Fatalf("expected ErrExist, got %v", err)
		}
		got, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("read existing: %v", err)
		}
		if string(got) != "keep" {
			t.Fatal("existing file overwritten")
		}
		assertNoTemps(t, dir)
	})

	t.Run("existing_directory_preserved", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "existing-dir")
		if err := os.Mkdir(target, 0700); err != nil {
			t.Fatal(err)
		}
		err := writeExportFileWith(target, payload, writeSyncClose, os.Link)
		if err == nil {
			t.Fatal("expected error for directory target")
		}
		info, err := os.Lstat(target)
		if err != nil {
			t.Fatalf("lstat target: %v", err)
		}
		if !info.IsDir() {
			t.Fatal("directory target must remain a directory")
		}
		assertNoTemps(t, dir)
	})

	t.Run("dangling_symlink_preserved", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "dangling.json")
		linkTarget := filepath.Join(dir, "missing")
		if err := os.Symlink(linkTarget, target); err != nil {
			t.Fatal(err)
		}
		err := writeExportFileWith(target, payload, writeSyncClose, os.Link)
		if err == nil || !errors.Is(err, os.ErrExist) {
			t.Fatalf("expected ErrExist, got %v", err)
		}
		got, err := os.Readlink(target)
		if err != nil {
			t.Fatalf("symlink removed: %v", err)
		}
		if got != linkTarget {
			t.Fatal("symlink target changed")
		}
		assertNoTemps(t, dir)
	})

	t.Run("publish_failure_leaves_no_final_or_temp", func(t *testing.T) {
		dir := t.TempDir()
		target := filepath.Join(dir, "pubfail.json")
		err := writeExportFileWith(target, payload, writeSyncClose, func(string, string) error {
			return fmt.Errorf("injected publish failure")
		})
		if err == nil {
			t.Fatal("expected error")
		}
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Fatal("final must be absent")
		}
		assertNoTemps(t, dir)
	})
}
