package embed

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const DefaultModel = "all-minilm"

type Generator struct {
	model        string
	httpClient   *http.Client
	httpEndpoint *url.URL
}

func (g *Generator) validate() error {
	if g.httpClient == nil {
		return errors.New("httpClient is required")
	}
	if g.httpEndpoint == nil {
		return errors.New("httpEndpoint is required")
	}
	if g.model == "" {
		return errors.New("model is required")
	}
	return nil
}

func NewGenerator(httpClient *http.Client, endpoint *url.URL, model string) (*Generator, error) {
	g := &Generator{
		httpClient:   httpClient,
		httpEndpoint: endpoint,
		model:        model,
	}
	if g.model == "" {
		g.model = DefaultModel
	}
	return g, g.validate()
}

func (g *Generator) Generate(ctx context.Context, text string) ([]float32, error) {
	ctx, span := otel.Tracer("app").Start(ctx, "Generator.Generate", trace.WithAttributes(
		attribute.String("model", g.model),
		attribute.String("httpEndpoint", g.httpEndpoint.String()),
	))
	defer span.End()

	bs, err := json.Marshal(EndpointRequest{
		Model:  g.model,
		Prompt: text,
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		g.httpEndpoint.String(),
		bytes.NewReader(bs),
	)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	res, err := g.httpClient.Do(req)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		span.RecordError(fmt.Errorf("unexpected status code: %s", res.Status))
		span.SetStatus(codes.Error, res.Status)
		return nil, fmt.Errorf("unexpected status code: %s", res.Status)
	}

	var response EndpointResponse
	if err := json.NewDecoder(res.Body).Decode(&response); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	return response.Embedding, nil
}

type EndpointRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EndpointResponse struct {
	Embedding []float32 `json:"embedding"`
}
