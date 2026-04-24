package ai

import (
	"fmt"
	"strings"

	"go.uber.org/zap"

	"wenDao/internal/model"
	"wenDao/internal/pkg/eino"
)

// VectorService 向量服务接口
type VectorService interface {
	VectorizeArticle(articleID int64, title, content, slug string) error
	DeleteArticleVector(articleID int64) error
	SearchArticles(query string, topK int) ([]ArticleChunk, error)
	VectorizeKnowledgeDocument(documentID int64, title, content string) error
	DeleteKnowledgeDocumentVector(documentID int64) error
}

// ArticleChunk 文章片段
type ArticleChunk struct {
	ArticleID int64
	ChunkID   string
	Content   string
	Score     float32
}

type vectorService struct {
	vectorStore eino.RedisVectorStore
	embedder    eino.Embedder
	logger      *zap.Logger
}

func NewVectorService(
	vectorStore eino.RedisVectorStore,
	embedder eino.Embedder,
	logger *zap.Logger,
) VectorService {
	return &vectorService{vectorStore: vectorStore, embedder: embedder, logger: logger}
}

func (s *vectorService) VectorizeArticle(articleID int64, title, content, slug string) error {
	s.logger.Info("Starting article vectorization", zap.Int64("article_id", articleID), zap.String("title", title))
	if err := s.DeleteArticleVector(articleID); err != nil {
		s.logger.Warn("Failed to delete old vectors during re-vectorization", zap.Int64("article_id", articleID), zap.Error(err))
	}
	chunks := s.chunkArticle(title, content)
	if len(chunks) == 0 {
		s.logger.Warn("Article has no valid chunks to vectorize", zap.Int64("article_id", articleID))
		return nil
	}
	embeddings, err := s.embedder.EmbedBatch(chunks)
	if err != nil {
		return fmt.Errorf("failed to embed chunks: %w", err)
	}
	if len(embeddings) != len(chunks) {
		return fmt.Errorf("embedding count mismatch: got %d, expected %d", len(embeddings), len(chunks))
	}
	vectorItems := make([]eino.VectorItem, 0, len(chunks))
	for i, chunk := range chunks {
		key := fmt.Sprintf("vec:article:%d:chunk:%d", articleID, i)
		vectorItems = append(vectorItems, eino.VectorItem{
			Key:    key,
			Vector: embeddings[i],
			Metadata: map[string]interface{}{
				"source_kind": "article",
				"source_id":   articleID,
				"article_id":  articleID,
				"chunk_index": i,
				"title":       title,
				"slug":        slug,
				"content":     chunk,
			},
		})
	}
	if err := s.vectorStore.UpsertBatch(vectorItems); err != nil {
		return fmt.Errorf("failed to store vectors: %w", err)
	}
	return nil
}

func (s *vectorService) VectorizeKnowledgeDocument(documentID int64, title, content string) error {
	if err := s.DeleteKnowledgeDocumentVector(documentID); err != nil {
		s.logger.Warn("Failed to delete old knowledge document vectors during re-vectorization", zap.Int64("document_id", documentID), zap.Error(err))
	}
	chunks := s.chunkArticle(title, content)
	if len(chunks) == 0 {
		return nil
	}
	embeddings, err := s.embedder.EmbedBatch(chunks)
	if err != nil {
		return fmt.Errorf("failed to embed knowledge document chunks: %w", err)
	}
	if len(embeddings) != len(chunks) {
		return fmt.Errorf("embedding count mismatch: got %d, expected %d", len(embeddings), len(chunks))
	}
	vectorItems := make([]eino.VectorItem, 0, len(chunks))
	for i, chunk := range chunks {
		key := fmt.Sprintf("vec:knowledge:%d:chunk:%d", documentID, i)
		vectorItems = append(vectorItems, eino.VectorItem{
			Key:    key,
			Vector: embeddings[i],
			Metadata: map[string]interface{}{
				"source_kind": "knowledge_document",
				"source_id":   documentID,
				"chunk_index": i,
				"title":       title,
				"content":     chunk,
				"status":      model.KnowledgeDocumentStatusApproved,
			},
		})
	}
	return s.vectorStore.UpsertBatch(vectorItems)
}

func (s *vectorService) DeleteArticleVector(articleID int64) error {
	pattern := fmt.Sprintf("vec:article:%d:chunk:*", articleID)
	if err := s.vectorStore.Delete(pattern); err != nil {
		return fmt.Errorf("failed to delete vectors: %w", err)
	}
	return nil
}

func (s *vectorService) DeleteKnowledgeDocumentVector(documentID int64) error {
	pattern := fmt.Sprintf("vec:knowledge:%d:chunk:*", documentID)
	if err := s.vectorStore.Delete(pattern); err != nil {
		return fmt.Errorf("failed to delete knowledge vectors: %w", err)
	}
	return nil
}

func (s *vectorService) SearchArticles(query string, topK int) ([]ArticleChunk, error) {
	queryVector, err := s.embedder.Embed(query)
	if err != nil {
		return nil, fmt.Errorf("failed to embed query: %w", err)
	}
	results, err := s.vectorStore.Search(queryVector, topK)
	if err != nil {
		return nil, fmt.Errorf("failed to search vectors: %w", err)
	}
	chunks := make([]ArticleChunk, 0, len(results))
	for _, result := range results {
		var articleID int64
		if val, ok := result.Metadata["article_id"]; ok {
			switch v := val.(type) {
			case int64:
				articleID = v
			case int:
				articleID = int64(v)
			case string:
				fmt.Sscanf(v, "%d", &articleID)
			case []byte:
				fmt.Sscanf(string(v), "%d", &articleID)
			}
		}
		content := ""
		if val, ok := result.Metadata["content"]; ok {
			switch v := val.(type) {
			case string:
				content = v
			case []byte:
				content = string(v)
			}
		}
		chunks = append(chunks, ArticleChunk{ArticleID: articleID, ChunkID: result.Key, Content: content, Score: result.Score})
	}
	return chunks, nil
}

func (s *vectorService) chunkArticle(title, content string) []string {
	const (
		targetChunkSize = 600
		minChunkSize    = 100
		overlapSize     = 50
	)
	fullText := title + "\n\n" + content
	paragraphs := strings.Split(fullText, "\n")
	chunks := make([]string, 0)
	var currentChunk strings.Builder
	currentLength := 0
	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		if len([]rune(para)) > targetChunkSize {
			if currentLength > 0 {
				chunks = append(chunks, currentChunk.String())
				currentChunk.Reset()
				currentLength = 0
			}
			runes := []rune(para)
			for i := 0; i < len(runes); i += targetChunkSize - overlapSize {
				end := i + targetChunkSize
				if end > len(runes) {
					end = len(runes)
				}
				chunks = append(chunks, string(runes[i:end]))
				if end == len(runes) {
					break
				}
			}
			continue
		}
		if currentLength+len([]rune(para)) > targetChunkSize && currentLength >= minChunkSize {
			chunks = append(chunks, currentChunk.String())
			lastRunes := []rune(currentChunk.String())
			overlapStart := len(lastRunes) - overlapSize
			if overlapStart < 0 {
				overlapStart = 0
			}
			currentChunk.Reset()
			currentChunk.WriteString(string(lastRunes[overlapStart:]))
			currentChunk.WriteString("\n")
			currentLength = len(lastRunes[overlapStart:])
		}
		currentChunk.WriteString(para)
		currentChunk.WriteString("\n")
		currentLength += len([]rune(para))
	}
	if currentLength >= minChunkSize {
		chunks = append(chunks, currentChunk.String())
	} else if currentLength > 0 && len(chunks) > 0 {
		chunks[len(chunks)-1] = chunks[len(chunks)-1] + "\n" + currentChunk.String()
	} else if currentLength > 0 {
		chunks = append(chunks, currentChunk.String())
	}
	return chunks
}
