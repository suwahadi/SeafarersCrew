package repository_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/suwahadi/seafarerscrew/backend/internal/platform/postgres"
)

// Shared containers keep the suite fast; each test cleans the tables it touches.
var (
	testDB    *pgxpool.Pool
	testRedis *goredis.Client
	testDSN   string
)

func TestMain(m *testing.M) {
	os.Exit(runWithContainers(m))
}

func runWithContainers(m *testing.M) int {
	if os.Getenv("SKIP_CONTAINER_TESTS") == "1" {
		fmt.Println("skipping repository tests: SKIP_CONTAINER_TESTS=1")
		return 0
	}
	ctx := context.Background()
	pg, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("seafarers_test"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(60*time.Second)),
	)
	if err != nil {
		fmt.Println("start postgres container:", err)
		return 1
	}
	defer func() { _ = testcontainers.TerminateContainer(pg) }()

	rc, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		fmt.Println("start redis container:", err)
		return 1
	}
	defer func() { _ = testcontainers.TerminateContainer(rc) }()

	testDSN, err = pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		fmt.Println("postgres dsn:", err)
		return 1
	}
	if err := postgres.Migrate(testDSN, "up"); err != nil {
		fmt.Println("migrate:", err)
		return 1
	}
	testDB, err = postgres.Open(ctx, testDSN)
	if err != nil {
		fmt.Println("open pool:", err)
		return 1
	}
	defer testDB.Close()

	redisURL, err := rc.ConnectionString(ctx)
	if err != nil {
		fmt.Println("redis url:", err)
		return 1
	}
	opts, err := goredis.ParseURL(redisURL)
	if err != nil {
		fmt.Println("parse redis url:", err)
		return 1
	}
	testRedis = goredis.NewClient(opts)
	defer func() { _ = testRedis.Close() }()

	return m.Run()
}

func truncate(t *testing.T, tables ...string) {
	t.Helper()
	for _, table := range tables {
		if _, err := testDB.Exec(context.Background(), "TRUNCATE "+table+" CASCADE"); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
}
