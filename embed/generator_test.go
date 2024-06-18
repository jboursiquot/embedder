package embed_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/jboursiquot/embedder/embed"
)

func TestGenerate(t *testing.T) {
	tests := map[string]struct {
		input        string
		expectedErr  bool
		mockResponse embed.EndpointResponse
	}{
		"valid response": {
			input:       "test prompt",
			expectedErr: false,
			mockResponse: embed.EndpointResponse{
				Embedding: []float32{0.1, 0.2, 0.3},
			},
		},
		"invalid response": {
			input:        "test prompt",
			expectedErr:  true,
			mockResponse: embed.EndpointResponse{},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			// Mock server to simulate the endpoint
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.expectedErr {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(tt.mockResponse) //nolint:errcheck
			}))
			defer mockServer.Close()

			// Parse mock server URL
			endpointURL, err := url.Parse(mockServer.URL)
			if err != nil {
				t.Fatalf("Failed to parse mock server URL: %v", err)
			}

			// Create the generator
			generator, err := embed.NewGenerator(mockServer.Client(), endpointURL, "")
			if err != nil {
				if !tt.expectedErr {
					t.Fatalf("Failed to create generator: %v", err)
				}
				return
			}

			// Test the Generate method
			ctx := context.Background()
			embedding, err := generator.Generate(ctx, tt.input)
			if err != nil {
				if !tt.expectedErr {
					t.Fatalf("Expected no error, got %v", err)
				}
				return
			}
			if tt.expectedErr {
				t.Fatalf("Expected error, got none")
			}

			// Validate the response
			expectedEmbedding := tt.mockResponse.Embedding
			if len(embedding) != len(expectedEmbedding) {
				t.Fatalf("Expected embedding length %d, got %d", len(expectedEmbedding), len(embedding))
			}
			for i := range embedding {
				if embedding[i] != expectedEmbedding[i] {
					t.Errorf("Expected embedding[%d] to be %f, got %f", i, expectedEmbedding[i], embedding[i])
				}
			}
		})
	}
}
