package server

import (
	"context"
	"errors"
	"io"
	"strings"

	pb "github.com/your-org/cortado/agent/gen/agent/v1"
	lspmanager "github.com/your-org/cortado/agent/internal/lsp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const lspLanguageMetadataKey = "x-cortado-lsp-language"

func (s *AgentServer) OpenLSP(ctx context.Context, req *pb.OpenLSPRequest) (*pb.OpenLSPResponse, error) {
	language := strings.TrimSpace(req.GetLanguage())
	if language == "" {
		return nil, status.Error(codes.InvalidArgument, "language is required")
	}

	server, err := s.lspMgr.GetOrStart(language)
	if err != nil {
		return nil, mapLSPError(err)
	}
	if server == nil {
		return nil, status.Error(codes.Internal, "lsp server did not start")
	}

	s.lspMu.Lock()
	s.lspLanguage = strings.ToLower(language)
	s.lspMu.Unlock()

	return &pb.OpenLSPResponse{}, nil
}

func (s *AgentServer) StreamLSP(stream pb.WorkspaceAgentService_StreamLSPServer) error {
	language, err := s.resolveLSPLanguage(stream.Context())
	if err != nil {
		return err
	}

	server, err := s.lspMgr.GetOrStart(language)
	if err != nil {
		return mapLSPError(err)
	}

	events, release, err := server.Attach()
	if err != nil {
		return mapLSPError(err)
	}
	defer release()

	ctx, cancel := context.WithCancel(stream.Context())
	defer cancel()

	errCh := make(chan error, 1)
	go s.pipeStreamToLSP(ctx, stream, server, errCh)

	for {
		select {
		case <-ctx.Done():
			if err := stream.Context().Err(); err != nil && status.Code(err) == codes.Canceled {
				return nil
			}
			return nil
		case event, ok := <-events:
			if !ok {
				return status.Error(codes.Unavailable, "lsp server exited")
			}
			if event.Err != nil {
				return status.Errorf(codes.Unavailable, "lsp server exited: %v", event.Err)
			}
			if err := stream.Send(&pb.LSPMessage{Data: event.Data}); err != nil {
				return err
			}
		case err := <-errCh:
			if err == nil {
				return nil
			}
			return err
		}
	}
}

func (s *AgentServer) pipeStreamToLSP(ctx context.Context, stream pb.WorkspaceAgentService_StreamLSPServer, server *lspmanager.Server, errCh chan<- error) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
				select {
				case errCh <- nil:
				case <-ctx.Done():
				}
				return
			}
			select {
			case errCh <- err:
			case <-ctx.Done():
			}
			return
		}

		if err := server.Write(msg.GetData()); err != nil {
			select {
			case errCh <- mapLSPError(err):
			case <-ctx.Done():
			}
			return
		}
	}
}

func (s *AgentServer) activeLSPLanguage() string {
	s.lspMu.RLock()
	defer s.lspMu.RUnlock()
	return s.lspLanguage
}

func (s *AgentServer) resolveLSPLanguage(ctx context.Context) (string, error) {
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		if values := md.Get(lspLanguageMetadataKey); len(values) > 0 {
			language := strings.TrimSpace(values[0])
			if language == "" {
				return "", status.Error(codes.InvalidArgument, "lsp language metadata must not be empty")
			}
			return strings.ToLower(language), nil
		}
	}

	languages := s.lspMgr.Languages()
	switch len(languages) {
	case 0:
		return "", status.Error(codes.FailedPrecondition, "open lsp before streaming")
	case 1:
		return languages[0], nil
	default:
		return "", status.Error(codes.InvalidArgument, "multiple lsp servers are active; specify x-cortado-lsp-language metadata")
	}
}

func mapLSPError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, lspmanager.ErrStreamAttached):
		return status.Error(codes.FailedPrecondition, "lsp stream already attached")
	case errors.Is(err, lspmanager.ErrServerClosed):
		return status.Error(codes.Unavailable, "lsp server closed")
	default:
		if strings.Contains(err.Error(), "unsupported language") || strings.Contains(err.Error(), "language is required") {
			return status.Error(codes.InvalidArgument, err.Error())
		}
		if strings.Contains(err.Error(), "find dart executable") || strings.Contains(err.Error(), "stat dart binary") {
			return status.Error(codes.FailedPrecondition, err.Error())
		}
		return status.Errorf(codes.Internal, "lsp operation failed: %v", err)
	}
}
