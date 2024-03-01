package sqltestutil

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestStartPostgresContainer(t *testing.T) {
	ctx := context.Background()

	container, err := StartPostgresContainer(ctx, "15")
	if err != nil {
		t.Fatalf("could not start container: %v", err)
	}
	defer container.Shutdown(ctx)

	db, err := sql.Open("pgx", container.ConnectionString())
	if err != nil {
		container.Shutdown(ctx)
		t.Fatalf("could not open connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		container.Shutdown(ctx)
		t.Fatalf("could not ping database: %v", err)
	}

	row := db.QueryRowContext(ctx, "SELECT NOW()")
	var result time.Time
	if err := row.Scan(&result); err != nil {
		container.Shutdown(ctx)
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
	defer container.Shutdown(ctx)

	// Get the connection string
	connStr := container.ConnectionString()

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		container.Shutdown(ctx)
		log.Fatalf("could not open connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		container.Shutdown(ctx)
		log.Fatalf("could not ping database: %v", err)
	}

	// Do something with the database
	row := db.QueryRowContext(ctx, "SELECT 2024")
	var result int
	if err := row.Scan(&result); err != nil {
		container.Shutdown(ctx)
		log.Fatalf("could not scan row: %v", err)
	}

	fmt.Println(result)
	// Output: 2024
}

func ExampleRunMigrationsAndScenario() {
	ctx := context.Background()

	// Create a new container
	container, err := StartPostgresContainer(ctx, "15")
	if err != nil {
		log.Fatalf("could not start container: %v", err)
	}
	defer container.Shutdown(ctx)

	db, err := sql.Open("pgx", container.ConnectionString())
	if err != nil {
		container.Shutdown(ctx)
		log.Fatalf("could not open connection: %v", err)
	}

	if err := db.Ping(); err != nil {
		container.Shutdown(ctx)
		log.Fatalf("could not ping database: %v", err)
	}

	// Run migrations
	err = RunMigrations(ctx, db, "testdata")
	if err != nil {
		container.Shutdown(ctx)
		log.Fatalf("could not run migrations: %v", err)
	}

	// Load a scenario
	err = LoadScenario(ctx, db, "testdata/scenario.yml")
	if err != nil {
		container.Shutdown(ctx)
		log.Fatalf("could not load scenario: %v", err)
	}

	// Do something with the database
	row := db.QueryRowContext(ctx, "SELECT username, password FROM users WHERE id = 1")
	var username, password string
	if err := row.Scan(&username, &password); err != nil {
		container.Shutdown(ctx)
		log.Fatalf("could not scan row: %v", err)
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
		container.Shutdown(ctx)
	}
}
