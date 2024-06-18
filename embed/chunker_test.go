package embed_test

import (
	"context"
	"strings"
	"testing"

	"github.com/jboursiquot/embedder/embed"
)

func TestChunker(t *testing.T) {
	tests := map[string]struct {
		input          string
		chunkSize      int
		chunkOverlap   int
		expectedChunks []string
	}{
		"simple": {
			input:        "This is a test string for chunking",
			chunkSize:    3,
			chunkOverlap: 1,
			expectedChunks: []string{
				"This is a",
				"a test string",
				"string for chunking",
				"chunking",
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			chunker, err := embed.NewChunker(tt.chunkSize, tt.chunkOverlap)
			if err != nil {
				t.Fatalf("Expected no error while creating chunker, got: %v", err)
			}

			reader := strings.NewReader(tt.input)
			ctx := context.Background()

			chunks, err := chunker.Chunk(ctx, reader)
			if err != nil {
				t.Fatalf("Expected no error while chunking, got: %v", err)
			}

			if len(chunks) != len(tt.expectedChunks) {
				t.Fatalf("Expected %d chunks, got: %d", len(tt.expectedChunks), len(chunks))
			}

			for i := range chunks {
				if chunks[i] != tt.expectedChunks[i] {
					t.Errorf("Expected chunk %d to be '%s', got: '%s'", i, tt.expectedChunks[i], chunks[i])
				}
			}
		})
	}
}
