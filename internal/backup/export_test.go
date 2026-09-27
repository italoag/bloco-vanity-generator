package backup

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
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
