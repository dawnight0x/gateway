package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"local-ai-gateway/internal/buildinfo"
	"local-ai-gateway/internal/store"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "gateway-backup:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return usageError()
	}
	if args[0] == "version" {
		info := buildinfo.Current()
		_, err := fmt.Fprintf(os.Stdout, "gateway-backup %s commit=%s built=%s\n", info.Version, info.Commit, info.BuiltAt)
		return err
	}
	if args[0] == "rotate-key" {
		return rotateKey(args[1:])
	}
	if args[0] != "restore" {
		return usageError()
	}
	flags := flag.NewFlagSet("restore", flag.ContinueOnError)
	input := flags.String("input", "", "portable backup archive")
	database := flags.String("database", "data/gateway.db", "restored database path")
	secret := flags.String("secret", "data/secret.key", "restored master key path")
	force := flags.Bool("force", false, "preserve and replace existing destination files")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if strings.TrimSpace(*input) == "" {
		return fmt.Errorf("--input is required")
	}
	passphrase := os.Getenv("GATEWAY_BACKUP_PASSPHRASE")
	if passphrase == "" {
		return fmt.Errorf("GATEWAY_BACKUP_PASSPHRASE is required")
	}
	return store.RestorePortableBackup(*input, *database, *secret, passphrase, *force)
}

func rotateKey(args []string) error {
	flags := flag.NewFlagSet("rotate-key", flag.ContinueOnError)
	database := flags.String("database", "data/gateway.db", "database path")
	secret := flags.String("secret", "data/secret.key", "master key path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	st, err := store.Open(*database, *secret)
	if err != nil {
		return err
	}
	ctx := context.Background()
	if _, err := st.CreateBackup(ctx, "pre-key-rotation"); err != nil {
		_ = st.Close()
		return fmt.Errorf("create pre-rotation snapshot: %w", err)
	}
	if err := st.RotateMasterKey(ctx); err != nil {
		_ = st.Close()
		return err
	}
	return st.Close()
}

func usageError() error {
	return fmt.Errorf("usage: gateway-backup restore --input <backup.zip> --database <gateway.db> --secret <secret.key> [--force] | gateway-backup rotate-key [--database <gateway.db>] [--secret <secret.key>]")
}
