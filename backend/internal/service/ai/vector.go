package ai

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/fnv"
	"math"
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

type ArticleSemanticProfileRepository interface {
	Upsert(profile *model.ArticleSemanticProfile) error
	DeleteByArticleID(articleID int64) error
}

// ArticleChunk 文章片段
type ArticleChunk struct {
	ArticleID int64
	ChunkID   string
	Content   string
	Score     float32
}

type vectorService struct {
	vectorStore         eino.RedisVectorStore
	embedder            eino.Embedder
	semanticProfileRepo ArticleSemanticProfileRepository
	logger              *zap.Logger
}

func NewVectorService(
	vectorStore eino.RedisVectorStore,
	embedder eino.Embedder,
	logger *zap.Logger,
	semanticProfileRepos ...ArticleSemanticProfileRepository,
) VectorService {
	var semanticProfileRepo ArticleSemanticProfileRepository
	if len(semanticProfileRepos) > 0 {
		semanticProfileRepo = semanticProfileRepos[0]
	}
	return &vectorService{
		vectorStore:         vectorStore,
		embedder:            embedder,
		semanticProfileRepo: semanticProfileRepo,
		logger:              logger,
	}
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
	if err := s.upsertArticleSemanticProfile(articleID, title, content, embeddings); err != nil {
		return fmt.Errorf("failed to store article semantic profile: %w", err)
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
	if s.semanticProfileRepo != nil {
		if err := s.semanticProfileRepo.DeleteByArticleID(articleID); err != nil {
			return fmt.Errorf("failed to delete article semantic profile: %w", err)
		}
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

func (s *vectorService) upsertArticleSemanticProfile(articleID int64, title, content string, embeddings [][]float32) error {
	if s.semanticProfileRepo == nil {
		return nil
	}
	articleVector := averageNormalizedVectors(embeddings)
	if len(articleVector) == 0 {
		return nil
	}
	x, y, z := projectSemanticVector(articleVector)
	profile := &model.ArticleSemanticProfile{
		ArticleID:    articleID,
		ContentHash:  articleSemanticContentHash(title, content),
		MapX:         x,
		MapY:         y,
		MapZ:         z,
		NeighborJSON: "[]",
	}
	if err := profile.SetEmbedding(articleVector); err != nil {
		return err
	}
	return s.semanticProfileRepo.Upsert(profile)
}

func averageNormalizedVectors(vectors [][]float32) []float32 {
	var sum []float64
	count := 0
	for _, vector := range vectors {
		normalized := normalizeFloat32Vector(vector)
		if len(normalized) == 0 {
			continue
		}
		if len(sum) == 0 {
			sum = make([]float64, len(normalized))
		}
		if len(normalized) != len(sum) {
			continue
		}
		for index, value := range normalized {
			sum[index] += float64(value)
		}
		count++
	}
	if count == 0 {
		return nil
	}
	average := make([]float32, len(sum))
	for index, value := range sum {
		average[index] = float32(value / float64(count))
	}
	return normalizeFloat32Vector(average)
}

func normalizeFloat32Vector(vector []float32) []float32 {
	if len(vector) == 0 {
		return nil
	}
	var norm float64
	for _, value := range vector {
		norm += float64(value) * float64(value)
	}
	if norm == 0 {
		return nil
	}
	scale := math.Sqrt(norm)
	normalized := make([]float32, len(vector))
	for index, value := range vector {
		normalized[index] = float32(float64(value) / scale)
	}
	return normalized
}

func projectSemanticVector(vector []float32) (float64, float64, float64) {
	axes := [3]float64{}
	for index, value := range vector {
		v := float64(value)
		axes[0] += v * semanticProjectionWeight(index, 0)
		axes[1] += v * semanticProjectionWeight(index, 1)
		axes[2] += v * semanticProjectionWeight(index, 2)
	}
	norm := math.Sqrt(axes[0]*axes[0] + axes[1]*axes[1] + axes[2]*axes[2])
	if norm == 0 {
		return 1, 0, 0
	}
	return axes[0] / norm, axes[1] / norm, axes[2] / norm
}

func semanticProjectionWeight(index int, axis int) float64 {
	hash := fnv.New64a()
	_, _ = hash.Write([]byte(fmt.Sprintf("article-semantic-map:%d:%d", axis, index)))
	bucket := int64(hash.Sum64() % 2000001)
	return float64(bucket-1000000) / 1000000
}

func articleSemanticContentHash(title, content string) string {
	sum := sha256.Sum256([]byte(title + "\n\n" + content))
	return hex.EncodeToString(sum[:])
}
