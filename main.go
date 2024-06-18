package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/jboursiquot/embedder/embed"
	"github.com/pgvector/pgvector-go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/trace"
	oteltrace "go.opentelemetry.io/otel/trace"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/plugin/opentelemetry/tracing"
)

var bookPath string
var db *gorm.DB

func init() {
	flag.StringVar(&bookPath, "book", "", "path to the book ")
	flag.Parse()

	var err error
	var dsn = "postgres://postgres:password@localhost:5432/test?sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:          logger.Default.LogMode(logger.Silent),
		CreateBatchSize: 1000,
	})
	if err != nil {
		log.Fatalln("failed to connect database", err)
	}

	if err := db.Use(tracing.NewPlugin(tracing.WithoutMetrics())); err != nil {
		log.Fatalln("failed to use tracing plugin", err)
	}

	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		log.Fatalln("failed to create extension", err)
	}

	if err := db.AutoMigrate(&embed.Book{}, &embed.BookEmbedding{}); err != nil {
		log.Fatalln("failed to migrate", err)
	}

	if err := db.Exec("CREATE INDEX ON book_embeddings USING hnsw (embedding vector_l2_ops)").Error; err != nil {
		log.Fatalln("failed to create index", err)
	}
}

func main() {
	ctx := context.Background()

	// Configure a new OTLP exporter using environment variables for sending data to Honeycomb over gRPC
	client := otlptracegrpc.NewClient()
	exp, err := otlptrace.New(ctx, client)
	if err != nil {
		log.Fatalf("failed to initialize exporter: %e", err)
	}

	// Create a new tracer provider with a batch span processor and the otlp exporter
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exp),
	)

	// Handle shutdown to ensure all sub processes are closed correctly and telemetry is exported
	defer func() {
		_ = exp.Shutdown(ctx)
		_ = tp.Shutdown(ctx)
	}()

	// Register the global Tracer provider
	otel.SetTracerProvider(tp)

	// Register the W3C trace context and baggage propagators so data is propagated across services/processes
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	tracer := otel.Tracer("app")
	ctx, span := tracer.Start(ctx, "main", oteltrace.WithAttributes(
		attribute.String("book", bookPath),
	))
	defer span.End()

	log.Println("Start")

	if bookPath == "" {
		log.Fatalln("book path is required")
	}

	f, err := os.Open(bookPath)
	if err != nil {
		log.Fatalln(err)
	}
	defer f.Close()

	var b embed.Book
	if err := db.WithContext(ctx).FirstOrCreate(
		&b,
		embed.Book{Title: "Meditations", Author: "Marcus Aurelius"},
	).Error; err != nil {
		log.Fatalln("failed to create book", err)
	}

	var endpoint *url.URL
	if endpoint, err = url.Parse("http://localhost:11434/api/embeddings"); err != nil {
		log.Fatalln(err)
	}

	chunker, err := embed.NewChunker(
		embed.DefaultChunkSize,
		embed.DefaultChunkOverlap,
	)
	if err != nil {
		log.Fatalln(err)
	}

	httpClient := http.Client{Timeout: 30 * time.Second}
	eg, err := embed.NewGenerator(&httpClient, endpoint, embed.DefaultModel)
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Chunking book...")

	chunks, err := chunker.Chunk(ctx, f)
	if err != nil {
		log.Fatalln(err)
	}

	bookEmbeddings := make([]embed.BookEmbedding, 0, len(chunks))

	for _, chunk := range chunks[:10] { // only process the first 10 chunks for now
		log.Println("Generating embeddings for chunk...")

		vals, err := eg.Generate(ctx, chunk)
		if err != nil {
			log.Fatalln(err)
		}

		bookEmbeddings = append(bookEmbeddings, embed.BookEmbedding{
			BookID:    b.ID,
			Text:      chunk,
			Embedding: pgvector.NewVector(vals),
		})
	}

	if err := db.WithContext(ctx).Save(&bookEmbeddings).Error; err != nil {
		log.Fatalln("failed to save book embeddings", err)
	}

	log.Println("Testing query for embeddings...")

	str := "How short lived the praiser and the praised, the one who remembers and the remembered."

	strEmbedding, err := eg.Generate(ctx, str)
	if err != nil {
		log.Fatalln(err)
	}

	var bookEmbeddingResults []embed.BookEmbedding
	db.WithContext(ctx).Clauses(
		clause.OrderBy{
			Expression: clause.Expr{
				SQL: "embedding <-> ?",
				Vars: []interface{}{
					pgvector.NewVector(strEmbedding),
				},
			},
		},
	).Limit(1).Find(&bookEmbeddingResults)

	for _, be := range bookEmbeddingResults {
		log.Println(be.Text[:100])
	}

	// ensure all spans are finished before the program exits
	span.End()
	tp.Shutdown(ctx) //nolint:errcheck
	log.Println("Done")
}
