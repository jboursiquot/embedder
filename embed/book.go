package embed

import (
	"github.com/pgvector/pgvector-go"
	"gorm.io/gorm"
)

type Book struct {
	gorm.Model
	Title      string
	Author     string
	Embeddings []BookEmbedding
}

type BookEmbedding struct {
	gorm.Model
	BookID    uint
	Text      string
	Embedding pgvector.Vector `gorm:"type:vector(384)"`
}
