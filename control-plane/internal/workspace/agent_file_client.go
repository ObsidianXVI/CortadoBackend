package workspace

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/cespare/xxhash/v2"
	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	defaultFileOperationTimeout = 5 * time.Minute
	fileTransferChunkSize       = 256 * 1024
)

type AgentFileServiceConfig struct {
	Dialer            func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	Logger            *log.Logger
	Timeout           time.Duration
	WorkspaceResolver ServiceResolver
}

type AgentFileService struct {
	dialer            func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error)
	timeout           time.Duration
	workspaceResolver ServiceResolver
}

func NewAgentFileService(cfg AgentFileServiceConfig) *AgentFileService {
	if cfg.Dialer == nil {
		cfg.Dialer = grpc.NewClient
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Default()
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = defaultFileOperationTimeout
	}

	return &AgentFileService{
		dialer:            cfg.Dialer,
		timeout:           cfg.Timeout,
		workspaceResolver: cfg.WorkspaceResolver,
	}
}

func (s *AgentFileService) ListDir(ctx context.Context, workspaceID, path string) ([]*agentpb.DirectoryEntry, error) {
	response, err := withAgentFileClient(ctx, s.timeout, s.dialer, s.workspaceResolver, workspaceID, func(callCtx context.Context, client agentpb.WorkspaceAgentServiceClient) ([]*agentpb.DirectoryEntry, error) {
		resp, err := client.ListDir(callCtx, &agentpb.ListDirRequest{Path: path})
		if err != nil {
			return nil, err
		}
		return resp.GetEntries(), nil
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *AgentFileService) ReadFile(ctx context.Context, workspaceID, path string, writer io.Writer) error {
	_, err := withAgentFileClient(ctx, s.timeout, s.dialer, s.workspaceResolver, workspaceID, func(callCtx context.Context, client agentpb.WorkspaceAgentServiceClient) (struct{}, error) {
		stream, err := client.ReadFile(callCtx, &agentpb.ReadFileRequest{Path: path})
		if err != nil {
			return struct{}{}, err
		}

		hasher := xxhash.New()
		var lastChunk *agentpb.ReadFileChunk
		for {
			response, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return struct{}{}, err
			}

			chunk := response.GetChunk()
			if chunk == nil {
				return struct{}{}, fmt.Errorf("%w: missing read file chunk", ErrInvalid)
			}
			if len(chunk.GetData()) > 0 {
				if _, err := writer.Write(chunk.GetData()); err != nil {
					return struct{}{}, err
				}
				if _, err := hasher.Write(chunk.GetData()); err != nil {
					return struct{}{}, fmt.Errorf("hash read file chunk: %w", err)
				}
			}
			lastChunk = chunk
		}

		if lastChunk == nil || !lastChunk.GetIsLast() {
			return struct{}{}, fmt.Errorf("%w: missing final read chunk", ErrInvalid)
		}
		if got, want := lastChunk.GetChecksum(), encodeXXHash64(hasher.Sum64()); !bytes.Equal(got, want) {
			return struct{}{}, fmt.Errorf("%w: read checksum mismatch", ErrInvalid)
		}

		return struct{}{}, nil
	})
	return err
}

func (s *AgentFileService) WriteFile(ctx context.Context, workspaceID, path string, reader io.Reader) (*agentpb.WriteFileResponse, error) {
	response, err := withAgentFileClient(ctx, s.timeout, s.dialer, s.workspaceResolver, workspaceID, func(callCtx context.Context, client agentpb.WorkspaceAgentServiceClient) (*agentpb.WriteFileResponse, error) {
		stream, err := client.WriteFile(callCtx)
		if err != nil {
			return nil, err
		}

		hasher := xxhash.New()
		buffer := make([]byte, fileTransferChunkSize)
		var seq int32

		for {
			n, readErr := reader.Read(buffer)
			if n > 0 {
				data := append([]byte(nil), buffer[:n]...)
				if _, err := hasher.Write(data); err != nil {
					return nil, fmt.Errorf("hash write chunk: %w", err)
				}

				req := &agentpb.WriteFileRequest{
					Chunk: &agentpb.WriteFileChunk{
						Path: path,
						Seq:  seq,
						Data: data,
					},
				}
				if readErr == io.EOF {
					req.Chunk.IsLast = true
					req.Chunk.Checksum = encodeXXHash64(hasher.Sum64())
				}
				if err := stream.Send(req); err != nil {
					return nil, err
				}
				seq++
			}

			if readErr == io.EOF {
				if n == 0 {
					if err := stream.Send(&agentpb.WriteFileRequest{
						Chunk: &agentpb.WriteFileChunk{
							Path:     path,
							Seq:      seq,
							IsLast:   true,
							Checksum: encodeXXHash64(hasher.Sum64()),
						},
					}); err != nil {
						return nil, err
					}
				}
				return stream.CloseAndRecv()
			}
			if readErr != nil {
				return nil, readErr
			}
		}
	})
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (s *AgentFileService) MakeDir(ctx context.Context, workspaceID, path string) error {
	_, err := withAgentFileClient(ctx, s.timeout, s.dialer, s.workspaceResolver, workspaceID, func(callCtx context.Context, client agentpb.WorkspaceAgentServiceClient) (struct{}, error) {
		_, err := client.MakeDir(callCtx, &agentpb.MakeDirRequest{Path: path})
		return struct{}{}, err
	})
	return err
}

func (s *AgentFileService) RenamePath(ctx context.Context, workspaceID, oldPath, newPath string) error {
	_, err := withAgentFileClient(ctx, s.timeout, s.dialer, s.workspaceResolver, workspaceID, func(callCtx context.Context, client agentpb.WorkspaceAgentServiceClient) (struct{}, error) {
		_, err := client.RenamePath(callCtx, &agentpb.RenamePathRequest{
			OldPath: oldPath,
			NewPath: newPath,
		})
		return struct{}{}, err
	})
	return err
}

func (s *AgentFileService) DeletePath(ctx context.Context, workspaceID, path string) error {
	_, err := withAgentFileClient(ctx, s.timeout, s.dialer, s.workspaceResolver, workspaceID, func(callCtx context.Context, client agentpb.WorkspaceAgentServiceClient) (struct{}, error) {
		_, err := client.DeletePath(callCtx, &agentpb.DeletePathRequest{Path: path})
		return struct{}{}, err
	})
	return err
}

func (s *AgentFileService) WatchFiles(ctx context.Context, workspaceID string, send func(*agentpb.FileEvent) error) error {
	_, err := withAgentFileClient(ctx, s.timeout, s.dialer, s.workspaceResolver, workspaceID, func(callCtx context.Context, client agentpb.WorkspaceAgentServiceClient) (struct{}, error) {
		stream, err := client.WatchFiles(callCtx, &agentpb.WatchFilesRequest{})
		if err != nil {
			return struct{}{}, err
		}

		for {
			response, err := stream.Recv()
			if err == io.EOF {
				return struct{}{}, nil
			}
			if err != nil {
				return struct{}{}, err
			}
			if event := response.GetEvent(); event != nil {
				if err := send(event); err != nil {
					return struct{}{}, err
				}
			}
		}
	})
	return err
}

func withAgentFileClient[T any](
	ctx context.Context,
	timeout time.Duration,
	dialer func(target string, opts ...grpc.DialOption) (*grpc.ClientConn, error),
	resolver ServiceResolver,
	workspaceID string,
	fn func(context.Context, agentpb.WorkspaceAgentServiceClient) (T, error),
) (T, error) {
	var zero T

	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	target := fmt.Sprintf("%s:%d", resolver.GetServiceDNS(workspaceID), defaultAgentIdleAddressPort)
	conn, err := dialer(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return zero, fmt.Errorf("dial workspace agent %q: %w", target, err)
	}
	defer conn.Close()

	result, err := fn(callCtx, agentpb.NewWorkspaceAgentServiceClient(conn))
	if err != nil {
		return zero, mapAgentFileError(err)
	}
	return result, nil
}

func mapAgentFileError(err error) error {
	if err == nil {
		return nil
	}

	if st, ok := status.FromError(err); ok {
		switch st.Code() {
		case codes.InvalidArgument:
			return fmt.Errorf("%w: %s", ErrInvalid, st.Message())
		case codes.AlreadyExists:
			return ErrAlreadyExists
		case codes.NotFound:
			return ErrNotFound
		}
	}

	return err
}

func encodeXXHash64(value uint64) []byte {
	checksum := make([]byte, 8)
	checksum[0] = byte(value >> 56)
	checksum[1] = byte(value >> 48)
	checksum[2] = byte(value >> 40)
	checksum[3] = byte(value >> 32)
	checksum[4] = byte(value >> 24)
	checksum[5] = byte(value >> 16)
	checksum[6] = byte(value >> 8)
	checksum[7] = byte(value)
	return checksum
}
