package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	defaultGeminiBaseURL       = "https://generativelanguage.googleapis.com/v1beta/models"
	defaultGeminiMaxOutput     = 256
	defaultGeminiModel         = "gemini-2.5-flash"
	defaultGeminiTemperature   = 0.2
	defaultGeminiScannerBuffer = 1024 * 1024
)

type GeminiProviderConfig struct {
	APIKey          string
	BaseURL         string
	HTTPClient      *http.Client
	MaxOutputTokens int
	Model           string
	Temperature     float64
}

type GeminiProvider struct {
	apiKey          string
	baseURL         string
	httpClient      *http.Client
	maxOutputTokens int
	model           string
	temperature     float64
}

func NewGeminiProvider(cfg GeminiProviderConfig) *GeminiProvider {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = defaultGeminiBaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{}
	}
	if cfg.MaxOutputTokens <= 0 {
		cfg.MaxOutputTokens = defaultGeminiMaxOutput
	}
	if strings.TrimSpace(cfg.Model) == "" {
		cfg.Model = defaultGeminiModel
	}
	if cfg.Temperature <= 0 {
		cfg.Temperature = defaultGeminiTemperature
	}

	return &GeminiProvider{
		apiKey:          strings.TrimSpace(cfg.APIKey),
		baseURL:         strings.TrimRight(cfg.BaseURL, "/"),
		httpClient:      cfg.HTTPClient,
		maxOutputTokens: cfg.MaxOutputTokens,
		model:           cfg.Model,
		temperature:     cfg.Temperature,
	}
}

func (p *GeminiProvider) StreamCompletion(ctx context.Context, prompt CompletionContext, emit func(string) error) error {
	if strings.TrimSpace(p.apiKey) == "" {
		return fmt.Errorf("%w: CORTADO_AI_API_KEY is not configured", ErrInvalidRequest)
	}

	requestBody := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": buildGeminiPrompt(prompt)},
				},
			},
		},
		"generationConfig": map[string]any{
			"maxOutputTokens": p.maxOutputTokens,
			"temperature":     p.temperature,
		},
	}

	body, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("marshal gemini request: %w", err)
	}

	endpoint := fmt.Sprintf("%s/%s:streamGenerateContent?alt=sse", p.baseURL, p.model)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build gemini request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	request.Header.Set("x-goog-api-key", p.apiKey)

	response, err := p.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("execute gemini request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode >= http.StatusBadRequest {
		detail, _ := io.ReadAll(io.LimitReader(response.Body, 16*1024))
		return fmt.Errorf("gemini request failed with status %d: %s", response.StatusCode, strings.TrimSpace(string(detail)))
	}

	return p.streamTokens(response.Body, emit)
}

func (p *GeminiProvider) streamTokens(body io.Reader, emit func(string) error) error {
	reader := bufio.NewReader(body)
	var eventData []string
	var emittedText string

	processEvent := func(lines []string) error {
		if len(lines) == 0 {
			return nil
		}

		payload := strings.TrimSpace(strings.Join(lines, "\n"))
		if payload == "" || payload == "[DONE]" {
			return nil
		}

		var chunk geminiStreamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			return fmt.Errorf("decode gemini stream chunk: %w", err)
		}
		text := chunk.text()
		if text == "" {
			if reason := chunk.blockReason(); reason != "" {
				return fmt.Errorf("gemini blocked prompt: %s", reason)
			}
			return nil
		}

		delta := textDelta(emittedText, text)
		if delta == "" {
			return nil
		}
		if err := emit(delta); err != nil {
			return err
		}
		emittedText += delta
		return nil
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return fmt.Errorf("read gemini stream: %w", err)
		}

		line = strings.TrimRight(line, "\r\n")
		switch {
		case line == "":
			if eventErr := processEvent(eventData); eventErr != nil {
				return eventErr
			}
			eventData = eventData[:0]
		case strings.HasPrefix(line, "data:"):
			eventData = append(eventData, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}

		if err == io.EOF {
			if eventErr := processEvent(eventData); eventErr != nil {
				return eventErr
			}
			return nil
		}
	}
}

type geminiStreamChunk struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	PromptFeedback struct {
		BlockReason string `json:"blockReason"`
	} `json:"promptFeedback"`
}

func (c geminiStreamChunk) text() string {
	var builder strings.Builder
	for _, candidate := range c.Candidates {
		for _, part := range candidate.Content.Parts {
			builder.WriteString(part.Text)
		}
	}
	return builder.String()
}

func (c geminiStreamChunk) blockReason() string {
	return strings.TrimSpace(c.PromptFeedback.BlockReason)
}

func textDelta(previous, current string) string {
	switch {
	case current == "":
		return ""
	case previous == "":
		return current
	case strings.HasPrefix(current, previous):
		return current[len(previous):]
	default:
		return current
	}
}

func buildGeminiPrompt(prompt CompletionContext) string {
	var builder strings.Builder
	builder.WriteString("You are Cortado's inline code completion engine.\n")
	builder.WriteString("Return only the text that should be inserted at the cursor.\n")
	builder.WriteString("Do not wrap the answer in markdown fences.\n")
	builder.WriteString("Do not repeat the provided prefix or suffix.\n")
	builder.WriteString("Keep the completion short and consistent with the surrounding file.\n\n")

	if prompt.Path != "" {
		builder.WriteString("File: ")
		builder.WriteString(prompt.Path)
		builder.WriteString("\n\n")
	}

	builder.WriteString("Code before cursor:\n<<<PREFIX\n")
	builder.WriteString(prompt.Prefix)
	builder.WriteString("\n>>>PREFIX\n\n")
	builder.WriteString("Code after cursor:\n<<<SUFFIX\n")
	builder.WriteString(prompt.Suffix)
	builder.WriteString("\n>>>SUFFIX\n")

	if len(prompt.Related) > 0 {
		builder.WriteString("\nRelated workspace context:\n")
		for index, chunk := range prompt.Related {
			builder.WriteString(fmt.Sprintf("[%d] %s", index+1, chunk.FilePath))
			if chunk.StartLine > 0 || chunk.EndLine > 0 {
				builder.WriteString(fmt.Sprintf(" (%d-%d)", chunk.StartLine, chunk.EndLine))
			}
			builder.WriteString("\n")
			builder.WriteString(chunk.Text)
			builder.WriteString("\n\n")
		}
	}

	if prompt.RetrievalQuery != "" {
		builder.WriteString("Semantic retrieval query:\n")
		builder.WriteString(prompt.RetrievalQuery)
		builder.WriteString("\n\n")
	}

	builder.WriteString("Respond with the completion text only.")
	return builder.String()
}
