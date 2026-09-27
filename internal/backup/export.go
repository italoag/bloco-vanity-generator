package backup

import (
	"context"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"bloco-vgen/internal/crypto"
)

func writeExportFile(path string, data []byte) error {
	return writeExportFileWith(path, data, writeSyncClose, os.Link)
}

func writeExportFileWith(path string, data []byte, write func(*os.File, []byte) error, publish func(string, string) error) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".export-tmp-*")
	if err != nil {
		return err
	}
	defer func() { _ = f.Close(); _ = os.Remove(f.Name()) }()
	if err := f.Chmod(0600); err != nil {
		return err
	}
	if err := write(f, data); err != nil {
		return err
	}
	if err := publish(f.Name(), path); err != nil {
		return err
	}
	return syncDir(filepath.Dir(path))
}

func Export(ctx context.Context, b *Bundle, outputDir string) (err error) {
	if err := ctx.Err(); err != nil {
		return err
	}
	if b == nil {
		return fmt.Errorf("bundle is nil")
	}
	if err := b.Validate(); err != nil {
		return err
	}
	if outputDir == "" || !filepath.IsAbs(outputDir) {
		return fmt.Errorf("output directory must be absolute")
	}
	parent := filepath.Dir(outputDir)
	if info, err := os.Lstat(parent); err != nil || !info.IsDir() {
		return fmt.Errorf("output parent directory invalid")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.Mkdir(outputDir, 0700); err != nil {
		return fmt.Errorf("output directory setup failed: %w", err)
	}
	createdInfo, statErr := os.Lstat(outputDir)
	if statErr != nil {
		return fmt.Errorf("output directory setup failed")
	}
	defer func() {
		if err == nil {
			return
		}
		current, statErr := os.Lstat(outputDir)
		if statErr != nil || !os.SameFile(createdInfo, current) {
			err = errors.Join(err, fmt.Errorf("export cleanup skipped: directory was replaced"))
			return
		}
		if rmErr := os.RemoveAll(outputDir); rmErr != nil {
			err = errors.Join(err, fmt.Errorf("export cleanup failed"))
		}
	}()

	payload, mErr := json.Marshal(b)
	if mErr != nil {
		err = fmt.Errorf("bundle serialization failed")
		return err
	}
	defer clear(payload)
	if len(payload) > MaxBundleBytes {
		err = fmt.Errorf("bundle too large")
		return err
	}
	if err = writeExportFile(filepath.Join(outputDir, "wallet-backup.json"), payload); err != nil {
		err = fmt.Errorf("export write failed")
		return err
	}
	if err = ctx.Err(); err != nil {
		return err
	}

	keyBytes, dErr := hex.DecodeString(strings.TrimPrefix(b.PrivateKey, "0x"))
	if dErr != nil {
		err = fmt.Errorf("invalid private key")
		return err
	}
	defer func() {
		for i := range keyBytes {
			keyBytes[i] = 0
		}
	}()

	switch b.Network {
	case "ethereum":
		service := crypto.NewKeyStoreService(crypto.KeyStoreConfig{
			Enabled:         true,
			OutputDirectory: outputDir,
		})
		if err = service.SaveWalletFilesToDisk(b.Address, b.Keystore, b.KeystorePassword, "ethereum", b.PrivateKey, b.Mnemonic); err != nil {
			err = fmt.Errorf("export write failed")
			return err
		}
	case "bitcoin":
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = writeExportFile(filepath.Join(outputDir, b.Address+".key"), []byte(b.PrivateKey)); err != nil {
			err = fmt.Errorf("export write failed")
			return err
		}
	case "solana":
		var keyArray [ed25519.PrivateKeySize]byte
		copy(keyArray[:], keyBytes)
		keyJSON, jErr := json.Marshal(keyArray)
		for i := range keyArray {
			keyArray[i] = 0
		}
		if jErr != nil {
			err = fmt.Errorf("export serialization failed")
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = writeExportFile(filepath.Join(outputDir, b.Address+".json"), keyJSON); err != nil {
			clear(keyJSON)
			err = fmt.Errorf("export write failed")
			return err
		}
		clear(keyJSON)
		if err = ctx.Err(); err != nil {
			return err
		}
		if err = writeExportFile(filepath.Join(outputDir, b.Address+".key"), []byte(b.PrivateKey)); err != nil {
			err = fmt.Errorf("export write failed")
			return err
		}
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	return nil
}
