package pgdump

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgsKeepThePasswordOutOfTheCommandLine(t *testing.T) {
	args, env, err := Args("postgres://seafarers:s3cret@db.internal:5433/crewdb?sslmode=require")
	require.NoError(t, err)
	assert.Contains(t, args, "--dbname=crewdb")
	assert.Contains(t, args, "--host=db.internal")
	assert.Contains(t, args, "--port=5433")
	assert.Contains(t, args, "--username=seafarers")
	assert.Contains(t, args, "--format=custom")
	for _, a := range args {
		assert.NotContains(t, a, "s3cret")
	}
	assert.ElementsMatch(t, []string{"PGPASSWORD=s3cret", "PGSSLMODE=require"}, env)
}

func TestArgsRejectUnusableURLs(t *testing.T) {
	for _, raw := range []string{"mysql://u:p@h/db", "postgres://u:p@h", "://bad"} {
		_, _, err := Args(raw)
		assert.Error(t, err, raw)
	}
}

func TestDumpStreamsOutputAndReportsFailures(t *testing.T) {
	dir := t.TempDir()
	ok := filepath.Join(dir, "ok.sh")
	require.NoError(t, os.WriteFile(ok, []byte("#!/bin/sh\nprintf 'DUMP:%s' \"$PGPASSWORD\"\n"), 0o700))
	var buf bytes.Buffer
	require.NoError(t, New(ok).Dump(context.Background(), "postgres://u:pw@h/db", &buf))
	assert.Equal(t, "DUMP:pw", buf.String())

	bad := filepath.Join(dir, "bad.sh")
	require.NoError(t, os.WriteFile(bad, []byte("#!/bin/sh\necho 'connection refused' >&2\nexit 1\n"), 0o700))
	err := New(bad).Dump(context.Background(), "postgres://u:pw@h/db", &bytes.Buffer{})
	assert.ErrorContains(t, err, "connection refused")
}
