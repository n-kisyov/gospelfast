package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gospelfast/gospelfast/internal/db"
	importpkg "github.com/gospelfast/gospelfast/internal/import"
	"github.com/gospelfast/gospelfast/internal/seed"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "gospelfast-cli",
		Short: "Gospelfast CLI import tool",
	}

	importCmd := &cobra.Command{
		Use:   "import",
		Short: "Import a Bible translation",
		RunE:  runImport,
	}

	importCmd.Flags().String("source", "", "Source file or URL (OSIS .xml, VPL .txt, or SWORD raw .zip)")
	importCmd.Flags().String("name", "", "Translation short name (e.g. KJV)")
	importCmd.Flags().String("format", "osis", "Input format: osis, rawzip, or vpl")
	importCmd.Flags().String("db-url", "postgres://gospelfast:gospelfast@localhost:5432/gospelfast?sslmode=disable", "PostgreSQL connection URL")
	importCmd.MarkFlagRequired("source")

	seedCmd := &cobra.Command{
		Use:   "seed",
		Short: "Download and import KJV from bible-api.com",
		RunE:  runSeed,
	}
	seedCmd.Flags().String("db-url", "postgres://gospelfast:gospelfast@localhost:5432/gospelfast?sslmode=disable", "PostgreSQL connection URL")

	rootCmd.AddCommand(importCmd, seedCmd)
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runImport(cmd *cobra.Command, args []string) error {
	source, _ := cmd.Flags().GetString("source")
	name, _ := cmd.Flags().GetString("name")
	format, _ := cmd.Flags().GetString("format")
	dbURL, _ := cmd.Flags().GetString("db-url")

	var f importpkg.Format
	switch strings.ToLower(format) {
	case "osis":
		f = importpkg.FormatOSIS
	case "rawzip":
		f = importpkg.FormatRawzip
	case "vpl":
		f = importpkg.FormatVPL
	default:
		return fmt.Errorf("unknown format: %s (use osis, rawzip, or vpl)", format)
	}

	ctx := context.Background()
	database, err := db.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer database.Close()

	pipeline := importpkg.New(database)

	progress := func(stage string, current, total int) {
		if total > 0 {
			pct := float64(current) / float64(total) * 100
			fmt.Printf("\r%s... %.1f%% (%d/%d)", stage, pct, current, total)
		} else {
			fmt.Printf("\r%s...", stage)
		}
	}

	return pipeline.ImportFile(ctx, source, name, f, progress)
}

func runSeed(cmd *cobra.Command, args []string) error {
	dbURL, _ := cmd.Flags().GetString("db-url")

	ctx := context.Background()
	database, err := db.New(ctx, dbURL)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer database.Close()

	pipeline := importpkg.New(database)

	progress := func(stage string, current, total int) {
		fmt.Printf("\r%s... %d/%d", stage, current, total)
	}

	fmt.Println("Downloading KJV from bible-api.com...")
	return seed.DownloadKJV(ctx, pipeline, progress)
}

func init() {
	log.SetOutput(os.Stderr)
}
