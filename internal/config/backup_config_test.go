package config

import (
	"testing"
	"time"
)

func TestBackupConfigDefaults(t *testing.T) {
	c := DefaultConfig()
	if c.Backup.Store != "files" {
		t.Fatalf("expected default store files, got %q", c.Backup.Store)
	}
	if c.Backup.OutputDir != "./backups" {
		t.Fatalf("expected default output dir ./backups, got %q", c.Backup.OutputDir)
	}
	if c.Backup.FnoxBinary != "fnox" {
		t.Fatalf("expected default fnox binary fnox, got %q", c.Backup.FnoxBinary)
	}
	if c.Backup.Timeout != 30*time.Second {
		t.Fatalf("expected default timeout 30s, got %v", c.Backup.Timeout)
	}
	if len(c.Backup.AgeRecipients) != 0 || c.Backup.AgeIdentityFile != "" ||
		c.Backup.KeychainService != "" || c.Backup.KeychainAccount != "" {
		t.Fatal("expected empty identity and keychain defaults")
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("default config failed validation: %v", err)
	}
}

func TestBackupConfigZeroValueAccepted(t *testing.T) {
	c := DefaultConfig()
	c.Backup = BackupConfig{}
	if err := c.Validate(); err != nil {
		t.Fatalf("zero-value backup config should be treated as legacy files: %v", err)
	}
}

func TestBackupConfigEnvironment(t *testing.T) {
	t.Setenv("BLOCO_BACKUP_STORE", " FNOX ")
	t.Setenv("BLOCO_BACKUP_DIR", "/tmp/backups")
	t.Setenv("BLOCO_FNOX_BINARY", "/opt/fnox")
	t.Setenv("BLOCO_AGE_RECIPIENTS", "age1aaa, age1bbb ,,age1ccc")
	t.Setenv("BLOCO_AGE_IDENTITY_FILE", "/tmp/identity.txt")

	c := DefaultConfig()
	c.LoadFromEnvironment()
	if c.Backup.Store != "fnox" {
		t.Fatalf("expected store fnox, got %q", c.Backup.Store)
	}
	if c.Backup.OutputDir != "/tmp/backups" || c.Backup.FnoxBinary != "/opt/fnox" {
		t.Fatalf("env mapping failed: %+v", c.Backup)
	}
	want := []string{"age1aaa", "age1bbb", "age1ccc"}
	if len(c.Backup.AgeRecipients) != len(want) {
		t.Fatalf("expected %d recipients, got %v", len(want), c.Backup.AgeRecipients)
	}
	for i := range want {
		if c.Backup.AgeRecipients[i] != want[i] {
			t.Fatalf("recipient %d mismatch: %q", i, c.Backup.AgeRecipients[i])
		}
	}
	if c.Backup.AgeIdentityFile != "/tmp/identity.txt" {
		t.Fatalf("identity file not mapped: %q", c.Backup.AgeIdentityFile)
	}
}

func TestBackupConfigEnvironmentPreservesDefaults(t *testing.T) {
	c := DefaultConfig()
	c.LoadFromEnvironment()
	if c.Backup.Store != "files" || c.Backup.FnoxBinary != "fnox" || c.Backup.Timeout != 30*time.Second {
		t.Fatalf("env-absent defaults not preserved: %+v", c.Backup)
	}
}

func TestBackupConfigValidation(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"files store ok", func(c *Config) { c.Backup.Store = "files" }, false},
		{"empty store ok", func(c *Config) { c.Backup.Store = "" }, false},
		{"invalid store", func(c *Config) { c.Backup.Store = "s3" }, true},
		{"negative timeout", func(c *Config) { c.Backup.Timeout = -time.Second }, true},
		{"zero timeout ok", func(c *Config) { c.Backup.Timeout = 0 }, false},
		{"fnox ok", func(c *Config) {
			c.Backup.Store = "fnox"
			c.Backup.AgeIdentityFile = "/tmp/id"
		}, false},
		{"fnox requires keystore", func(c *Config) {
			c.Backup.Store = "fnox"
			c.KeyStore.Enabled = false
		}, true},
		{"fnox requires output dir", func(c *Config) {
			c.Backup.Store = "fnox"
			c.Backup.OutputDir = ""
		}, true},
		{"identity conflicts service", func(c *Config) {
			c.Backup.Store = "fnox"
			c.Backup.AgeIdentityFile = "/tmp/id"
			c.Backup.KeychainService = "svc"
		}, true},
		{"identity conflicts account", func(c *Config) {
			c.Backup.Store = "fnox"
			c.Backup.AgeIdentityFile = "/tmp/id"
			c.Backup.KeychainAccount = "acct"
		}, true},
		{"keychain pair ok", func(c *Config) {
			c.Backup.Store = "fnox"
			c.Backup.KeychainService = "svc"
			c.Backup.KeychainAccount = "acct"
		}, false},
		{"keychain conflicts skipped for files", func(c *Config) {
			c.Backup.Store = "files"
			c.Backup.AgeIdentityFile = "/tmp/id"
			c.Backup.KeychainService = "svc"
		}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := DefaultConfig()
			tc.mutate(c)
			err := c.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected validation error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
