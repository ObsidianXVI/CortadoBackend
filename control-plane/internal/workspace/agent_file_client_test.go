package workspace

import (
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMapAgentFileError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "invalid argument",
			err:  status.Error(codes.InvalidArgument, "bad path"),
			want: ErrInvalid,
		},
		{
			name: "already exists",
			err:  status.Error(codes.AlreadyExists, "exists"),
			want: ErrAlreadyExists,
		},
		{
			name: "not found",
			err:  status.Error(codes.NotFound, "missing"),
			want: ErrPathNotFound,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := mapAgentFileError(test.err)
			if !errors.Is(got, test.want) {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}
