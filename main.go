// web serves the AStA website.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/astawh/web/internal/config"
	"github.com/astawh/web/internal/database"
	"github.com/astawh/web/model/users"
)

// Version and Revision are injected at build time.
var Version = "dev"
var Revision string

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	cfg := config.New()
	flags := flag.NewFlagSet("web", flag.ContinueOnError)
	addr := flags.String("addr", "127.0.0.1:8090", "HTTP listen address")
	version := flags.Bool("version", false, "Print application version")
	path := flags.String("database", cfg.Database, "Writable SQLite database path")
	bootstrap := flags.String("bootstrap-admin", "", "Create the first administrator with this email and exit")
	name := flags.String("bootstrap-name", "Administrator", "First administrator name")
	passwordFile := flags.String("bootstrap-password-file", "", "Private file containing the first administrator password")
	migrateOnly := flags.Bool("migrate-only", false, "Apply database migrations and exit")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected arguments: %v", flags.Args())
	}
	if *version {
		fmt.Println(Version)
		return nil
	}
	cfg.Database = *path
	if *bootstrap != "" || *migrateOnly {
		db, err := database.Open(ctx, cfg.Database)
		if err != nil {
			return err
		}
		defer db.Close()
		if *bootstrap != "" {
			info, err := os.Stat(*passwordFile)
			if err != nil {
				return errors.New("provide a private password file with -bootstrap-password-file")
			}
			if !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 || info.Size() > 1024 {
				return errors.New("password file must be regular, private (0600) and at most 1024 bytes")
			}
			password, err := os.ReadFile(*passwordFile)
			if err != nil {
				return err
			}
			if _, err := users.New(db).Bootstrap(ctx, *name, *bootstrap, strings.TrimRight(string(password), "\r\n")); err != nil {
				return err
			}
			log.Print("First administrator created")
		}
		return nil
	}
	app, err := newApplication(ctx, cfg)
	if err != nil {
		return err
	}
	defer app.Close()
	server := &http.Server{
		Addr: *addr, Handler: app.handler, ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 120 * time.Second,
	}
	done := make(chan error, 1)
	go func() { done <- server.ListenAndServe() }()
	log.Printf("AStA web %s: http://%s", Version, *addr)
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
			return err
		}
		if err := <-done; !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}
