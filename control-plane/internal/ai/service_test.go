package ai

import (
	"context"
	"strings"
	"testing"
)

func TestServiceAssemblesTrimmedContextAndRetrieval(t *testing.T) {
	provider := &providerStub{}
	embedder := &embedderStub{vector: []float64{0.1, 0.2}}
	retriever := &retrieverStub{
		chunks: []RetrievedChunk{
			{FilePath: "lib/helper.dart", StartLine: 10, EndLine: 14, Text: "String greet() => 'hi';"},
		},
	}
	service := NewService(ServiceConfig{
		MaxPrefixBytes: 32,
		MaxSuffixBytes: 16,
		MaxResults:     3,
		QueryLines:     5,
		Provider:       provider,
		QueryEmbedder:  embedder,
		Retriever:      retriever,
	})

	prefix := strings.Repeat("ignore\n", 20) + "line1\nline2\nline3\nline4\nline5"
	suffix := "abcdefghijklmnopqrst"
	var streamed strings.Builder

	err := service.StreamCompletion(context.Background(), CompletionParams{
		WorkspaceID: "ws-123",
		Path:        "lib/main.dart",
		Prefix:      prefix,
		Suffix:      suffix,
	}, func(token string) error {
		streamed.WriteString(token)
		return nil
	})
	if err != nil {
		t.Fatalf("stream completion: %v", err)
	}

	if embedder.query != "line1\nline2\nline3\nline4\nline5" {
		t.Fatalf("unexpected retrieval query: %q", embedder.query)
	}
	if retriever.workspaceID != "ws-123" {
		t.Fatalf("unexpected retriever workspace id: %q", retriever.workspaceID)
	}
	if provider.context.Path != "lib/main.dart" {
		t.Fatalf("unexpected provider path: %q", provider.context.Path)
	}
	if len([]byte(provider.context.Prefix)) > 32 {
		t.Fatalf("prefix was not trimmed: %d", len([]byte(provider.context.Prefix)))
	}
	if len([]byte(provider.context.Suffix)) > 16 {
		t.Fatalf("suffix was not trimmed: %d", len([]byte(provider.context.Suffix)))
	}
	if len(provider.context.Related) != 1 || provider.context.Related[0].FilePath != "lib/helper.dart" {
		t.Fatalf("unexpected related chunks: %#v", provider.context.Related)
	}
	if streamed.String() != "hello world" {
		t.Fatalf("unexpected streamed content: %q", streamed.String())
	}
}

func TestTextDeltaHandlesCumulativeAndIncrementalChunks(t *testing.T) {
	if got := textDelta("", "hel"); got != "hel" {
		t.Fatalf("unexpected empty previous delta: %q", got)
	}
	if got := textDelta("hel", "hello"); got != "lo" {
		t.Fatalf("unexpected cumulative delta: %q", got)
	}
	if got := textDelta("hello", " world"); got != " world" {
		t.Fatalf("unexpected incremental delta: %q", got)
	}
}

type providerStub struct {
	context CompletionContext
}

func (p *providerStub) StreamCompletion(_ context.Context, prompt CompletionContext, emit func(string) error) error {
	p.context = prompt
	if err := emit("hello"); err != nil {
		return err
	}
	return emit(" world")
}

type embedderStub struct {
	query  string
	vector []float64
}

func (e *embedderStub) EmbedQuery(_ context.Context, query string) ([]float64, error) {
	e.query = query
	return e.vector, nil
}

type retrieverStub struct {
	chunks      []RetrievedChunk
	workspaceID string
}

func (r *retrieverStub) Search(_ context.Context, workspaceID string, _ []float64, _ int) ([]RetrievedChunk, error) {
	r.workspaceID = workspaceID
	return r.chunks, nil
}
