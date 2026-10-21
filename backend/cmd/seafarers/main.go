// Command seafarers runs the API server and administrative CLI tasks.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/suwahadi/seafarerscrew/backend/internal/app"
	"github.com/suwahadi/seafarerscrew/backend/internal/config"
	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
	"github.com/suwahadi/seafarerscrew/backend/internal/support/apperr"
)

const usage = `Usage: seafarers <command> [flags]

Commands:
  serve                                   Run the HTTP API
  migrate up|down                         Apply all migrations or roll back one step
  db:seed [--demo] [--force]              Load reference data (idempotent); --demo adds demo data, --force allows it in production
  admin:create --name=<n> --email=<e>     Create the administrator (prints recovery codes)
  admin:reset-password --email=<e>        Print a temporary password and force a change
  backup:run                              Dump the database to object storage now
  healthcheck                             Probe the local /healthz endpoint (container healthcheck)
`

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "error:", describe(err))
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.Join(write(out, usage), errors.New("missing command"))
	}
	if args[0] == "healthcheck" {
		return healthcheck(ctx)
	}
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	switch args[0] {
	case "migrate":
		direction := "up"
		if len(args) > 1 {
			direction = args[1]
		}
		if err := postgres.Migrate(cfg.DatabaseURL, direction); err != nil {
			return err
		}
		return write(out, "migrate "+direction+": done\n")
	case "serve", "db:seed", "admin:create", "admin:reset-password", "backup:run":
		return withApp(ctx, cfg, func(a *app.App) error { return dispatch(ctx, a, args, out) })
	default:
		return errors.Join(write(out, usage), fmt.Errorf("unknown command %q", args[0]))
	}
}

func withApp(ctx context.Context, cfg config.Config, fn func(a *app.App) error) error {
	a, err := app.New(ctx, cfg, app.NewLogger(cfg.LogLevel))
	if err != nil {
		return err
	}
	defer a.Close()
	return fn(a)
}

func dispatch(ctx context.Context, a *app.App, args []string, out io.Writer) error {
	switch args[0] {
	case "serve":
		return a.Serve(ctx)
	case "db:seed":
		return seed(ctx, a, args[1:], out)
	case "admin:create":
		return createAdmin(ctx, a, args[1:], out)
	case "backup:run":
		return runBackup(ctx, a, out)
	default:
		return resetPassword(ctx, a, args[1:], out)
	}
}

func seed(ctx context.Context, a *app.App, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("db:seed", flag.ContinueOnError)
	demo := fs.Bool("demo", false, "also load the demo data set")
	force := fs.Bool("force", false, "allow demo data when APP_ENV is production")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *force && !*demo {
		return errors.New("--force only applies together with --demo")
	}
	steps, err := a.Seed(ctx, app.SeedOptions{Demo: *demo, Force: *force})
	if err != nil {
		return err
	}
	if !*demo {
		return write(out, "reference data seeded\n")
	}
	loaded, skipped := 0, 0
	for _, s := range steps {
		if s.Skipped {
			skipped++
		} else {
			loaded++
		}
	}
	return write(out, fmt.Sprintf("reference data seeded; demo data: %d steps loaded, %d already present\n", loaded, skipped))
}

func runBackup(ctx context.Context, a *app.App, out io.Writer) error {
	b, err := a.Backups.RunNow(ctx)
	if err != nil {
		return err
	}
	return write(out, fmt.Sprintf("backup %s: %s (%d bytes)\n", b.Status, b.StorageKey, b.SizeBytes))
}

func createAdmin(ctx context.Context, a *app.App, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("admin:create", flag.ContinueOnError)
	name := fs.String("name", "", "administrator name")
	email := fs.String("email", "", "administrator email")
	pass := fs.String("password", "", "initial password (optional; generated when empty)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	res, err := a.Auth.CreateAdmin(ctx, *name, *email, *pass)
	if err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Administrator created: %s <%s>\n", res.Admin.Name, res.Admin.Email)
	if res.GeneratedPassword != "" {
		fmt.Fprintf(&b, "Temporary password (change it at first login): %s\n", res.GeneratedPassword)
	}
	b.WriteString("\nRecovery codes (shown once; store them offline):\n")
	for _, c := range res.RecoveryCodes {
		b.WriteString("  " + c + "\n")
	}
	return write(out, b.String())
}

func resetPassword(ctx context.Context, a *app.App, args []string, out io.Writer) error {
	fs := flag.NewFlagSet("admin:reset-password", flag.ContinueOnError)
	email := fs.String("email", "", "administrator email")
	if err := fs.Parse(args); err != nil {
		return err
	}
	temp, err := a.Auth.ResetPassword(ctx, *email)
	if err != nil {
		return err
	}
	return write(out, fmt.Sprintf("Temporary password: %s\nAll sessions were revoked; a password change is required at next login.\n", temp))
}

func healthcheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://127.0.0.1:8080/healthz", nil)
	if err != nil {
		return fmt.Errorf("build healthcheck request: %w", err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("healthcheck: %w", err)
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("healthcheck: status %d", res.StatusCode)
	}
	return nil
}

func write(out io.Writer, s string) error {
	if _, err := io.WriteString(out, s); err != nil {
		return fmt.Errorf("write output: %w", err)
	}
	return nil
}

func describe(err error) string {
	e, ok := apperr.As(err)
	if !ok || len(e.Fields) == 0 {
		return err.Error()
	}
	parts := make([]string, 0, len(e.Fields))
	for k, v := range e.Fields {
		parts = append(parts, k+": "+v)
	}
	return e.Message + " " + strings.Join(parts, "; ")
}
