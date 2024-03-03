package sqltestutil

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"testing"
)

func TestRunMigrations(t *testing.T) {
	t.Parallel()

	type args struct {
		db           ExecerContext
		migrationDir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "good",
			args: args{
				db:           &mockExecerContext{debug: true},
				migrationDir: "testdata",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := RunMigrations(context.Background(), tt.args.db, tt.args.migrationDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("RunMigrations() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

type mockExecerContext struct {
	hasError bool
	debug    bool
}

func (m *mockExecerContext) ExecContext(
	ctx context.Context,
	query string,
	args ...interface{},
) (sql.Result, error) {
	if m.debug {
		log.Printf("executing query: %s [%+v]", query, args)
	}

	if m.hasError {
		return nil, errors.New("error")
	}

	return nil, nil
}
