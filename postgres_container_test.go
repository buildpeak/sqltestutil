package sqltestutil

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	leak := flag.Bool("leak", false, "leak the container")
	flag.Parse()

	if *leak {
		goleak.VerifyTestMain(m)

		return
	}

	// Setup

	// Run tests
	exitCode := m.Run()

	// Teardown

	os.Exit(exitCode)
}

func TestStartPostgresContainer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	container, err := StartPostgresContainer(ctx, "15")
	if err != nil {
		t.Fatalf("could not start container: %v", err)
	}

	t.Cleanup(func() {
		// Cleanup the container
		_ = container.Shutdown(ctx)
	})

	db, err := sql.Open("pgx", container.ConnectionString())
	if err != nil {
		t.Fatalf("could not open connection: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	if err := db.Ping(); err != nil {
		t.Fatalf("could not ping database: %v", err)
	}

	row := db.QueryRowContext(ctx, "SELECT NOW()")
	var result time.Time
	if err := row.Scan(&result); err != nil {
		t.Fatalf("could not scan row: %v", err)
	}

	t.Logf("result: %s", result)
}

func ExamplePostgresContainer() {
	ctx := context.Background()

	// Create a new container
	container, err := StartPostgresContainer(ctx, "15")
	if err != nil {
		log.Fatalf("could not start container: %v", err)
	}

	defer func() {
		// Cleanup the container
		_ = container.Shutdown(ctx)
	}()

	// Get the connection string
	connStr := container.ConnectionString()

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Printf("could not open connection: %v", err)
		return
	}

	defer func() {
		_ = db.Close()
	}()

	if err := db.Ping(); err != nil {
		log.Printf("could not ping database: %v", err)
		return
	}

	// Do something with the database
	row := db.QueryRowContext(ctx, "SELECT 2024")
	var result int
	if err := row.Scan(&result); err != nil {
		log.Printf("could not scan row: %v", err)
	}

	fmt.Println(result)
	// Output: 2024
}

func ExampleRunMigrations() {
	ctx := context.Background()

	// Create a new container
	container, err := StartPostgresContainer(ctx, "15")
	if err != nil {
		log.Fatalf("could not start container: %v", err)
	}
	defer func() { _ = container.Shutdown(ctx) }()

	db, err := sql.Open("pgx", container.ConnectionString())
	if err != nil {
		log.Printf("could not open connection: %v", err)
		return
	}

	defer func() { _ = db.Close() }()

	if err := db.Ping(); err != nil {
		log.Printf("could not ping database: %v", err)
		return
	}

	// Run migrations
	err = RunMigrations(ctx, db, "testdata")
	if err != nil {
		log.Printf("could not run migrations: %v", err)
	}

	// Load a scenario
	err = LoadScenario(ctx, db, "testdata/scenario.yml")
	if err != nil {
		log.Printf("could not load scenario: %v", err)
		return
	}

	// Do something with the database
	row := db.QueryRowContext(ctx, "SELECT username, password FROM users WHERE id = 1")
	var username, password string
	if err := row.Scan(&username, &password); err != nil {
		log.Printf("could not scan row: %v", err)
		return
	}

	fmt.Printf("%s %s\n", username, password)
	// Output: user1 password1
}

func BenchmarkStartPostgresContainer(b *testing.B) {
	ctx := context.Background()
	for i := 0; i < b.N; i++ {
		container, err := StartPostgresContainer(ctx, "15")
		if err != nil {
			b.Fatalf("could not start container: %v", err)
		}
		_ = container.Shutdown(ctx)
	}
}
