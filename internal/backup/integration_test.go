package backup

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func integrationCmd(ctx context.Context, name string, env []string, cwd string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = cwd
	cmd.Env = env
	return cmd
}

func TestFnoxIntegration(t *testing.T) {
	if os.Getenv("BLOCO_FNOX_INTEGRATION") != "1" {
		t.Skip("set BLOCO_FNOX_INTEGRATION=1 to run")
	}
	fnoxBin, err := exec.LookPath("fnox")
	if err != nil {
		t.Fatal("fnox binary required for integration test")
	}
	ageKeygen, err := exec.LookPath("age-keygen")
	if err != nil {
		t.Fatal("age-keygen binary required for integration test")
	}

	home := t.TempDir()
	baseEnv := []string{"HOME=" + home, "PATH=" + os.Getenv("PATH"), "NO_COLOR=1"}

	verCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	out, err := integrationCmd(verCtx, fnoxBin, baseEnv, home, "--version").Output()
	cancel()
	if err != nil {
		t.Fatalf("fnox --version failed: %v", err)
	}
	if strings.TrimSpace(string(out)) != "fnox "+SupportedFnoxVersion {
		t.Fatalf("unsupported fnox version: %s", strings.TrimSpace(string(out)))
	}

	identityFile := filepath.Join(home, "identity.txt")
	kgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	if _, err := integrationCmd(kgCtx, ageKeygen, baseEnv, home, "-o", identityFile).CombinedOutput(); err != nil {
		cancel()
		t.Fatalf("age-keygen failed: %v", err)
	}
	cancel()
	pubCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	pub, err := integrationCmd(pubCtx, ageKeygen, baseEnv, home, "-y", identityFile).Output()
	cancel()
	if err != nil {
		t.Fatalf("age-keygen -y failed: %v", err)
	}
	recipient := strings.TrimSpace(string(pub))
	if !validAgeRecipient(recipient) {
		t.Fatal("unexpected recipient format")
	}

	outDir := t.TempDir()
	if err := os.Chmod(outDir, 0700); err != nil {
		t.Fatal(err)
	}
	store, err := NewStore(Options{
		Binary:       fnoxBin,
		OutputDir:    outDir,
		Recipients:   []string{recipient},
		IdentityFile: identityFile,
		Timeout:      30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := store.Doctor(ctx); err != nil {
		t.Fatalf("Doctor failed: %v", err)
	}

	bundles := map[string]*Bundle{
		"ethereum": ethereumBundle(t, true),
		"bitcoin":  bitcoinBundle(t, true),
		"solana":   solanaBundle(t),
	}
	for name, b := range bundles {
		t.Run(name, func(t *testing.T) {
			receipt, err := store.Save(ctx, b)
			if err != nil {
				t.Fatalf("Save failed: %v", err)
			}
			loaded, err := store.Load(ctx, receipt.Path)
			if err != nil {
				t.Fatalf("Load failed: %v", err)
			}
			if loaded.PrivateKey != b.PrivateKey || loaded.Address != b.Address || loaded.ID != b.ID {
				t.Fatal("loaded bundle mismatch")
			}
			data, err := os.ReadFile(receipt.Path)
			if err != nil {
				t.Fatal(err)
			}
			for _, s := range []string{b.PrivateKey, b.Mnemonic, b.KeystorePassword} {
				if s != "" && strings.Contains(string(data), s) {
					t.Fatal("plaintext secret in artifact")
				}
			}
			exportDir := filepath.Join(t.TempDir(), "export")
			if err := Export(ctx, loaded, exportDir); err != nil {
				t.Fatalf("Export failed: %v", err)
			}
			if name == "solana" {
				arrData, err := os.ReadFile(filepath.Join(exportDir, loaded.Address+".json"))
				if err != nil {
					t.Fatal(err)
				}
				var arr []int
				if err := json.Unmarshal(arrData, &arr); err != nil || len(arr) != 64 || len(arrData) == 0 || arrData[0] != '[' {
					t.Fatal("solana keypair export is not a 64-element array")
				}
				want, _ := hex.DecodeString(loaded.PrivateKey)
				for i, v := range arr {
					if v < 0 || v > 255 || byte(v) != want[i] {
						t.Fatal("keypair element mismatch")
					}
				}
			}
		})
	}

	t.Run("moved_copy_new_store", func(t *testing.T) {
		b := bundles["ethereum"]
		src := filepath.Join(outDir, b.Filename())
		raw, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		movedHome := t.TempDir()
		t.Setenv("HOME", movedHome)
		identityCopy := filepath.Join(movedHome, "identity.txt")
		identityRaw, err := os.ReadFile(identityFile)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(identityCopy, identityRaw, 0600); err != nil {
			t.Fatal(err)
		}
		moved := filepath.Join(t.TempDir(), b.Filename())
		if err := os.WriteFile(moved, raw, 0600); err != nil {
			t.Fatal(err)
		}
		movedStore, err := NewStore(Options{
			Binary:       fnoxBin,
			OutputDir:    t.TempDir(),
			Recipients:   []string{recipient},
			IdentityFile: identityCopy,
			Timeout:      30 * time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := movedStore.Load(ctx, moved)
		if err != nil {
			t.Fatalf("Load of moved copy failed: %v", err)
		}
		if loaded.PrivateKey != b.PrivateKey {
			t.Fatal("moved copy mismatch")
		}
	})

	t.Run("wrong_identity", func(t *testing.T) {
		wrongIdentity := filepath.Join(home, "wrong.txt")
		kgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		if _, err := integrationCmd(kgCtx, ageKeygen, baseEnv, home, "-o", wrongIdentity).CombinedOutput(); err != nil {
			cancel()
			t.Fatalf("age-keygen failed: %v", err)
		}
		cancel()
		wrongStore, err := NewStore(Options{
			Binary:       fnoxBin,
			OutputDir:    t.TempDir(),
			Recipients:   []string{recipient},
			IdentityFile: wrongIdentity,
			Timeout:      30 * time.Second,
		})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := wrongStore.Load(ctx, filepath.Join(outDir, bundles["ethereum"].Filename())); err == nil {
			t.Fatal("expected decryption failure with wrong identity")
		}
	})

	t.Run("collision_existing", func(t *testing.T) {
		if _, err := store.Save(ctx, bundles["ethereum"]); err == nil {
			t.Fatal("expected collision: artifact already exists")
		}
	})
}
