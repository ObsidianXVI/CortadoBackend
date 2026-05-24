package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	pb "github.com/your-org/cortado/agent/gen/agent/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const snapshotTag = "stop"

type snapshotCommandRunner func(ctx context.Context, env []string, name string, args ...string) ([]byte, error)

func (s *AgentServer) CreateSnapshot(ctx context.Context, req *pb.CreateSnapshotRequest) (*pb.CreateSnapshotResponse, error) {
	if strings.TrimSpace(s.snapshotBucket) == "" || strings.TrimSpace(s.snapshotPassword) == "" || strings.TrimSpace(s.workspaceID) == "" {
		return nil, status.Error(codes.FailedPrecondition, "snapshot repository is not configured")
	}

	repository := fmt.Sprintf("gs:%s:/%s", s.snapshotBucket, s.workspaceID)
	env := []string{"RESTIC_PASSWORD=" + s.snapshotPassword}

	if err := s.ensureSnapshotRepository(ctx, env, repository); err != nil {
		return nil, err
	}
	if _, err := s.commandRunner(
		ctx,
		env,
		"restic",
		"-r",
		repository,
		"backup",
		s.workspaceRoot,
		"--tag",
		snapshotTag,
		"--exclude",
		".git",
		"--exclude",
		"node_modules",
	); err != nil {
		return nil, mapSnapshotError("create workspace snapshot", err)
	}

	return &pb.CreateSnapshotResponse{Repository: repository}, nil
}

func (s *AgentServer) ensureSnapshotRepository(ctx context.Context, env []string, repository string) error {
	output, err := s.commandRunner(ctx, env, "restic", "-r", repository, "init")
	if err == nil || snapshotRepositoryAlreadyExists(output) {
		return nil
	}

	return mapSnapshotError("init snapshot repository", err)
}

func mapSnapshotError(operation string, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		return status.Errorf(codes.DeadlineExceeded, "%s timed out", operation)
	default:
		return status.Errorf(codes.Internal, "%s: %v", operation, err)
	}
}

func snapshotRepositoryAlreadyExists(output []byte) bool {
	trimmed := string(bytes.TrimSpace(output))
	if trimmed == "" {
		return false
	}

	for _, marker := range []string{
		"already initialized",
		"config file already exists",
	} {
		if strings.Contains(trimmed, marker) {
			return true
		}
	}

	return false
}

func runSnapshotCommand(ctx context.Context, env []string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = append(os.Environ(), env...)

	output, err := cmd.CombinedOutput()
	if err == nil {
		return output, nil
	}
	if len(output) == 0 {
		return nil, err
	}

	return output, fmt.Errorf("%w: %s", err, bytes.TrimSpace(output))
}
