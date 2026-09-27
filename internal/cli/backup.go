package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"bloco-vgen/internal/backup"
	"bloco-vgen/internal/crypto"
	"bloco-vgen/internal/worker"
	"bloco-vgen/pkg/wallet"
)

type fnoxBackupStore interface {
	Doctor(context.Context) error
	Save(context.Context, *backup.Bundle) (backup.Receipt, error)
	Load(context.Context, string) (*backup.Bundle, error)
}

func (app *Application) newFnoxStore() (*backup.Store, error) {
	outputDir, err := filepath.Abs(app.config.Backup.OutputDir)
	if err != nil {
		return nil, fmt.Errorf("invalid backup directory")
	}
	identityFile := app.config.Backup.AgeIdentityFile
	if identityFile != "" {
		identityFile, err = filepath.Abs(identityFile)
		if err != nil {
			return nil, fmt.Errorf("invalid age identity file")
		}
	}
	return backup.NewStore(backup.Options{
		Binary:          app.config.Backup.FnoxBinary,
		Recipients:      app.config.Backup.AgeRecipients,
		IdentityFile:    identityFile,
		KeychainService: app.config.Backup.KeychainService,
		KeychainAccount: app.config.Backup.KeychainAccount,
		OutputDir:       outputDir,
		Timeout:         app.config.Backup.Timeout,
	})
}

func (app *Application) ensureFnoxStore() error {
	if app.backupStore != nil {
		return nil
	}
	store, err := app.newFnoxStore()
	if err != nil {
		return err
	}
	app.backupStore = store
	return nil
}

func (app *Application) backupReceipt(w *wallet.Wallet) (backup.Receipt, bool) {
	key := strings.ToLower(w.Network) + "/" + w.Address
	v, ok := app.backupReceipts.Load(key)
	if !ok {
		return backup.Receipt{}, false
	}
	return v.(backup.Receipt), true
}

func (app *Application) saveFnoxWallet(ctx context.Context, w *wallet.Wallet) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := app.ensureFnoxStore(); err != nil {
		return err
	}

	network := strings.ToLower(w.Network)
	if network == "" {
		network = "ethereum"
	}

	var keystoreV3 *crypto.KeyStoreV3
	var keystorePassword string
	if network == "ethereum" {
		keystoreService, _, err := app.buildKeyStoreService()
		if err != nil {
			return err
		}
		keystoreV3, keystorePassword, err = keystoreService.GenerateKeyStore(w.PrivateKey, w.Address, network)
		if err != nil {
			return fmt.Errorf("keystore generation failed for address %s", w.Address)
		}
	}

	b, err := backup.NewBundle(w, keystoreV3, keystorePassword)
	if err != nil {
		return fmt.Errorf("invalid wallet bundle: %w", err)
	}

	receipt, err := app.backupStore.Save(ctx, b)
	if err != nil {
		return err
	}
	app.backupReceipts.Store(strings.ToLower(w.Network)+"/"+w.Address, receipt)
	return nil
}

func (app *Application) displayPrivateKey(key string) string {
	if app.config.Backup.Store == "fnox" {
		return "[encrypted backup]"
	}
	return key
}

func (app *Application) displayFnoxBackupResults(results []*wallet.GenerationResult) {
	if app.config.Backup.Store != "fnox" {
		return
	}
	for i, result := range results {
		if result == nil || result.Wallet == nil {
			continue
		}
		fmt.Printf("Wallet %d: %s %s\n", i+1, result.Wallet.Network, result.Wallet.Address)
		if receipt, ok := app.backupReceipt(result.Wallet); ok {
			fmt.Printf("  Encrypted backup confirmed: %q\n", receipt.Path)
		} else {
			fmt.Printf("  Encrypted backup not confirmed\n")
		}
	}
}

func (app *Application) createBackupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage encrypted fnox wallet backups",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "doctor",
		Short: "Verify the fnox backup provider is functional",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := app.parseFlags(cmd); err != nil {
				return err
			}
			if err := app.ensureFnoxStore(); err != nil {
				return err
			}
			if err := app.backupStore.Doctor(cmd.Context()); err != nil {
				return err
			}
			cmd.Println("Fnox backup provider verified")
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "verify <file>",
		Short: "Verify an encrypted backup file decrypts and is valid",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := app.parseFlags(cmd); err != nil {
				return err
			}
			if err := app.ensureFnoxStore(); err != nil {
				return err
			}
			b, err := app.backupStore.Load(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			cmd.Printf("Network: %s\n", b.Network)
			cmd.Printf("Address: %s\n", b.Address)
			cmd.Printf("ID: %s\n", b.ID)
			if b.MnemonicRole == "unrelated" {
				cmd.Println("Warning: stored mnemonic is unrelated to the private key and cannot restore it")
			}
			cmd.Println("Backup verified")
			return nil
		},
	})

	exportCmd := &cobra.Command{
		Use:   "export <file>",
		Short: "Export an encrypted backup to plaintext wallet files",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputDir, _ := cmd.Flags().GetString("output-dir")
			allowPlaintext, _ := cmd.Flags().GetBool("allow-plaintext")
			if outputDir == "" {
				return fmt.Errorf("--output-dir is required")
			}
			if !allowPlaintext {
				return fmt.Errorf("export writes plaintext secrets; re-run with --allow-plaintext to acknowledge")
			}
			if err := app.parseFlags(cmd); err != nil {
				return err
			}
			if err := app.ensureFnoxStore(); err != nil {
				return err
			}
			absDir, err := filepath.Abs(outputDir)
			if err != nil {
				return fmt.Errorf("invalid output directory")
			}
			b, err := app.backupStore.Load(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if err := backup.Export(cmd.Context(), b, absDir); err != nil {
				return err
			}
			cmd.Printf("WARNING: exported files at %q contain plaintext secrets; protect or delete them after use\n", absDir)
			return nil
		},
	}
	exportCmd.Flags().String("output-dir", "", "New directory for exported plaintext files (must not exist)")
	exportCmd.Flags().Bool("allow-plaintext", false, "Acknowledge that export writes plaintext secrets")
	cmd.AddCommand(exportCmd)

	return cmd
}

func (app *Application) generateMultipleWalletsFnoxText(ctx context.Context, workerPool worker.WorkerPool, criteria wallet.GenerationCriteria, count int, showProgress bool) error {
	start := time.Now()
	var totalAttempts int64
	if showProgress && !app.config.CLI.QuietMode {
		fmt.Printf("Generating %d wallets with encrypted backups\n", count)
	}
	for i := 0; i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		result, err := workerPool.GenerateWalletWithContext(ctx, criteria)
		if err != nil {
			return fmt.Errorf("wallet generation interrupted: %w", err)
		}
		if err := app.generateAndSaveKeystoreContext(ctx, result.Wallet, app.config.CLI.VerboseOutput); err != nil {
			fmt.Printf("Wallet %d: %s %s\nEncrypted backup not confirmed\n", i+1, result.Wallet.Network, result.Wallet.Address)
			return fmt.Errorf("failed to persist wallet %s: %w", result.Wallet.Address, err)
		}
		receipt, ok := app.backupReceipt(result.Wallet)
		if !ok {
			return fmt.Errorf("encrypted backup receipt missing")
		}
		totalAttempts += result.Attempts
		fmt.Printf("Wallet %d: %s %s\nEncrypted backup confirmed: %q\n", i+1, result.Wallet.Network, result.Wallet.Address, receipt.Path)
	}
	fmt.Printf("Encrypted backups confirmed: %d/%d\nTotal attempts: %s\nTotal duration: %s\n", count, count, formatLargeNumber(totalAttempts), formatDuration(time.Since(start)))
	return nil
}
