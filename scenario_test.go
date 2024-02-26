package sqltestutil

import (
	"context"
	"testing"
)

func TestLoadScenario(t *testing.T) {
	type args struct {
		ctx      context.Context
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
				ctx:      context.Background(),
				db:       &mockExecerContext{debug: true},
				filename: "testdata/scenario.yml",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := LoadScenario(tt.args.ctx, tt.args.db, tt.args.filename); (err != nil) != tt.wantErr {
				t.Errorf("LoadScenario() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
