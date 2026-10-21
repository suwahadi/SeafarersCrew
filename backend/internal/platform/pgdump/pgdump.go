// Package pgdump runs pg_dump against the application database.
package pgdump

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// Runner invokes the pg_dump binary.
type Runner struct {
	binary string
}

// New returns a Runner; an empty binary means "pg_dump" from PATH.
func New(binary string) *Runner {
	if binary == "" {
		binary = "pg_dump"
	}
	return &Runner{binary: binary}
}

// Args returns the arguments and environment for dumping databaseURL; the password only ever goes into the environment.
func Args(databaseURL string) ([]string, []string, error) {
	u, err := url.Parse(databaseURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse database url: %w", err)
	}
	if u.Scheme != "postgres" && u.Scheme != "postgresql" {
		return nil, nil, errors.New("database url must use the postgres scheme")
	}
	db := strings.TrimPrefix(u.Path, "/")
	if db == "" {
		return nil, nil, errors.New("database url has no database name")
	}
	args := []string{"--format=custom", "--compress=9", "--no-owner", "--no-privileges", "--dbname=" + db}
	var env []string
	if h := u.Hostname(); h != "" {
		args = append(args, "--host="+h)
	}
	if p := u.Port(); p != "" {
		args = append(args, "--port="+p)
	}
	if name := u.User.Username(); name != "" {
		args = append(args, "--username="+name)
	}
	if pw, ok := u.User.Password(); ok {
		env = append(env, "PGPASSWORD="+pw)
	}
	if mode := u.Query().Get("sslmode"); mode != "" {
		env = append(env, "PGSSLMODE="+mode)
	}
	return args, env, nil
}

// Dump writes a custom-format, compressed dump of the database to w.
func (r *Runner) Dump(ctx context.Context, databaseURL string, w io.Writer) error {
	args, env, err := Args(databaseURL)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, r.binary, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = w
	var stderr strings.Builder
	cmd.Stderr = &limitedWriter{w: &stderr, max: 4096}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

type limitedWriter struct {
	w   *strings.Builder
	max int
}

func (l *limitedWriter) Write(p []byte) (int, error) {
	if room := l.max - l.w.Len(); room > 0 {
		l.w.Write(p[:min(room, len(p))])
	}
	return len(p), nil
}
