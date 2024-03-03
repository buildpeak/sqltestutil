package sqltestutil

import (
	"context"
	"testing"
)

func TestLoadScenario(t *testing.T) {
	t.Parallel()

	type args struct {
		db       ExecerContext
		filename string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "good",
			args: args{
				db:       &mockExecerContext{debug: true},
				filename: "testdata/scenario.yml",
			},
		},
	}
	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if err := LoadScenario(context.Background(), tt.args.db, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf(
					"LoadScenario() error = %v, wantErr %v",
					err,
					tt.wantErr,
				)
			}
		})
	}
}
