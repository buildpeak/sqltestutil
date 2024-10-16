package sqltestutil

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// ExecerContext is an interface used by MustExecContext and LoadFileContext
type ExecerContext interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// RunMigrations reads all of the files matching *.up.sql in migrationDir and
// executes them in lexicographical order against the provided db. A typical
// convention is to use a numeric prefix for each new migration, e.g.:
//
//	001_create_users.up.sql
//	002_create_posts.up.sql
//	003_create_comments.up.sql
//
// Note that this function does not check whether the migration has already been
// run. Its primary purpose is to initialize a test database.
func RunMigrations(ctx context.Context, db ExecerContext, migrationDir string) error {
	filenames, err := filepath.Glob(filepath.Join(migrationDir, "*.up.sql"))
	if err != nil {
		return fmt.Errorf("glob migrationDir error: %w", err)
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		data, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("read file error: %w", err)
		}
		_, err = db.ExecContext(ctx, string(data))
		if err != nil {
			return fmt.Errorf("exec file error: %w", err)
		}
	}
	return nil
}
