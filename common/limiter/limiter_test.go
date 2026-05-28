package limiter

import (
	"errors"
	"testing"
)

func TestIsNoScriptError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "redis noscript",
			err:  errors.New("NOSCRIPT No matching script. Please use EVAL."),
			want: true,
		},
		{
			name: "other redis error",
			err:  errors.New("WRONGTYPE Operation against a key holding the wrong kind of value"),
			want: false,
		},
		{
			name: "nil",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNoScriptError(tt.err); got != tt.want {
				t.Fatalf("isNoScriptError() = %v, want %v", got, tt.want)
			}
		})
	}
}
