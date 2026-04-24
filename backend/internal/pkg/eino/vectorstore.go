package eino

import (
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// RedisVectorStore Redis 向量存储接口
type RedisVectorStore interface {
	InitIndex(indexName string, dim int) error
	Upsert(key string, vector []float32, metadata map[string]interface{}) error
	UpsertBatch(items []VectorItem) error
	Delete(pattern string) error
	Search(vector []float32, topK int) ([]SearchResult, error)
}

// VectorItem 向量数据项
type VectorItem struct {
	Key      string
	Vector   []float32
	Metadata map[string]interface{}
}

// SearchResult 搜索结果
type SearchResult struct {
	Key      string
	Score    float32
	Metadata map[string]interface{}
}

// redisVectorStore Redis 向量存储实现
type redisVectorStore struct {
	client    *redis.Client
	indexName string
	logger    *zap.Logger
}

// NewRedisVectorStore 创建 Redis 向量存储实例
func NewRedisVectorStore(client *redis.Client, indexName string, logger *zap.Logger) RedisVectorStore {
	return &redisVectorStore{
		client:    client,
		indexName: indexName,
		logger:    logger,
	}
}

// InitIndex 初始化向量索引
func (s *redisVectorStore) InitIndex(indexName string, dim int) error {
	ctx := context.Background()

	// 1. 检查索引是否存在
	_, err := s.client.Do(ctx, "FT.INFO", s.indexName).Result()
	if err == nil {
		s.logger.Info("[VectorStore] Index already exists", zap.String("index", s.indexName))
		return nil
	}

	s.logger.Info("[VectorStore] Creating new index",
		zap.String("index", s.indexName),
		zap.Int("dim", dim),
		zap.String("prefix", "vec:"))

	// 使用最稳健的创建语法
	cmd := []interface{}{
		"FT.CREATE", s.indexName,
		"ON", "HASH",
		"PREFIX", "1", "vec:",
		"SCHEMA",
		"embedding", "VECTOR", "HNSW", "6",
		"TYPE", "FLOAT32",
		"DIM", dim,
		"DISTANCE_METRIC", "COSINE",
		"source_kind", "TAG",
		"source_id", "NUMERIC",
		"article_id", "NUMERIC",
		"chunk_index", "NUMERIC",
		"title", "TEXT",
		"slug", "TEXT",
		"content", "TEXT",
	}

	if _, err = s.client.Do(ctx, cmd...).Result(); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	return nil
}

// UpsertBatch 批量插入/更新向量
func (s *redisVectorStore) UpsertBatch(items []VectorItem) error {
	ctx := context.Background()
	pipe := s.client.Pipeline()
	for _, item := range items {
		fields := make(map[string]interface{})
		fields["embedding"] = float32SliceToBytes(item.Vector)
		for k, v := range item.Metadata {
			// 强制转为字符串，确保 Redis 存储格式一致
			switch val := v.(type) {
			case int, int32, int64:
				fields[k] = fmt.Sprintf("%d", val)
			case float32, float64:
				fields[k] = fmt.Sprintf("%f", val)
			default:
				fields[k] = v
			}
		}
		pipe.HSet(ctx, item.Key, fields)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *redisVectorStore) Upsert(key string, vector []float32, metadata map[string]interface{}) error {
	return s.UpsertBatch([]VectorItem{{Key: key, Vector: vector, Metadata: metadata}})
}

func (s *redisVectorStore) Delete(pattern string) error {
	ctx := context.Background()
	var cursor uint64
	for {
		keys, nextCursor, err := s.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			s.client.Del(ctx, keys...)
		}
		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}
	return nil
}

// Search 向量搜索
func (s *redisVectorStore) Search(vector []float32, topK int) ([]SearchResult, error) {
	ctx := context.Background()
	vectorBytes := float32SliceToBytes(vector)

	// 使用 DIALECT 2 的语法
	query := fmt.Sprintf("(*)=>[KNN %d @embedding $query_vec AS __embedding_score]", topK)
	cmd := []interface{}{
		"FT.SEARCH", s.indexName,
		query,
		"PARAMS", "2", "query_vec", vectorBytes,
		"SORTBY", "__embedding_score", "ASC",
		"RETURN", "8", "__embedding_score", "source_kind", "source_id", "article_id", "chunk_index", "title", "slug", "content",
		"DIALECT", "2",
	}

	s.logger.Info("[VectorStore] Executing Search",
		zap.String("index", s.indexName),
		zap.String("query", query),
		zap.Int("input_dim", len(vector)))

	result, err := s.client.Do(ctx, cmd...).Result()
	if err != nil {
		s.logger.Error("[VectorStore] FT.SEARCH Failed", zap.Error(err))
		return nil, err
	}

	s.logger.Info("[VectorStore] Raw search result",
		zap.String("type", fmt.Sprintf("%T", result)))

	var results []SearchResult
	switch v := result.(type) {
	case []interface{}:
		results = parseSearchResults(v)
	case map[interface{}]interface{}:
		s.logger.Info("[VectorStore] Handling map result",
			zap.Any("keys", getMapKeys(v)),
			zap.Any("total_results", v["total_results"]),
			zap.Any("warning", v["warning"]))

		rawResults, ok := v["results"].([]interface{})
		if !ok {
			s.logger.Warn("[VectorStore] Map result does not contain 'results' key or it is not a slice")
			return []SearchResult{}, nil
		}

		// 如果已经是 Map 模式，v["results"] 里的内容通常不再包含 total 计数
		// 而是直接按 [doc1_id, [fields...], doc2_id, [fields...]] 或者直接就是结构化对象
		// 这里我们先尝试用 parseStructuredResults 来处理
		results = parseStructuredResults(rawResults)
	default:
		s.logger.Warn("[VectorStore] Search result is an unexpected type", zap.String("type", fmt.Sprintf("%T", result)))
		return []SearchResult{}, nil
	}

	s.logger.Info("[VectorStore] Search Completed", zap.Int("found", len(results)))

	for i, res := range results {
		content := "N/A"
		if c, ok := res.Metadata["content"]; ok {
			content = toString(c)
		}
		if len(content) > 150 {
			content = content[:150] + "..."
		}
		s.logger.Info(fmt.Sprintf("[VectorStore] Result %d", i+1),
			zap.String("key", res.Key),
			zap.Float32("score", res.Score),
			zap.String("content", content))
	}

	return results, nil
}

func parseStructuredResults(items []interface{}) []SearchResult {
	results := make([]SearchResult, 0)
	// 根据日志，items 的每个元素本身就是一个 map [id:..., extra_attributes:...]
	// 而不是传统的 [key, fields, key, fields] 扁平切片

	for _, item := range items {
		m, ok := item.(map[interface{}]interface{})
		if !ok {
			// 如果不是 map，尝试按照扁平切片逻辑处理（备选）
			// 但基于日志，这里应该是一个完整的对象
			continue
		}

		key := toString(m["id"])
		if key == "" {
			// 尝试从其他字段获取 key
			key = toString(m["key"])
		}

		var metadata map[string]interface{}
		var score float32

		// 核心修复：从 extra_attributes 中提取元数据
		if attrs, ok := m["extra_attributes"].(map[interface{}]interface{}); ok {
			metadata, score = parseFieldsMap(attrs)
		} else {
			// 备选方案：直接解析 m
			metadata, score = parseFieldsMap(m)
		}

		results = append(results, SearchResult{
			Key:      key,
			Score:    score,
			Metadata: metadata,
		})
	}
	return results
}

func parseFieldsSlice(fields []interface{}) (map[string]interface{}, float32) {
	metadata := make(map[string]interface{})
	var score float32
	for j := 0; j < len(fields); j += 2 {
		if j+1 >= len(fields) {
			break
		}
		name := toString(fields[j])
		val := fields[j+1]
		if name == "__embedding_score" {
			score = toFloat32(val)
			continue
		}
		metadata[name] = val
	}
	return metadata, score
}

func parseFieldsMap(fields map[interface{}]interface{}) (map[string]interface{}, float32) {
	metadata := make(map[string]interface{})
	var score float32
	for k, v := range fields {
		name := toString(k)
		if name == "__embedding_score" {
			score = toFloat32(v)
			continue
		}
		metadata[name] = v
	}
	return metadata, score
}

func getMapKeys(m map[interface{}]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, fmt.Sprintf("%v", k))
	}
	return keys
}

func parseSearchResults(resultSlice []interface{}) []SearchResult {
	results := make([]SearchResult, 0)
	if len(resultSlice) < 1 {
		return results
	}

	total := toInt64(resultSlice[0])
	if total == 0 {
		return results
	}

	for i := 1; i < len(resultSlice); i += 2 {
		if i+1 >= len(resultSlice) {
			break
		}

		key := toString(resultSlice[i])
		fields, ok := resultSlice[i+1].([]interface{})
		if !ok {
			continue
		}

		metadata := make(map[string]interface{})
		var score float32

		for j := 0; j < len(fields); j += 2 {
			if j+1 >= len(fields) {
				break
			}
			name := toString(fields[j])
			val := fields[j+1]

			if name == "__embedding_score" {
				score = toFloat32(val)
				continue
			}
			metadata[name] = val
		}

		results = append(results, SearchResult{
			Key:      key,
			Score:    score,
			Metadata: metadata,
		})

		// 增加内容预览日志
		contentPreview := "N/A"
		if c, ok := metadata["content"]; ok {
			switch v := c.(type) {
			case string:
				contentPreview = v
			case []byte:
				contentPreview = string(v)
			}
		}
		if len(contentPreview) > 100 {
			contentPreview = contentPreview[:100] + "..."
		}
		// 这里由于函数是私有的，暂时使用全局格式或通过 context 传递 logger，
		// 但为了简单，我们先在 parse 阶段保留 metadata，在外层 Search 中统一打印。
	}
	return results
}

func toString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return string(val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	case []byte:
		i, _ := strconv.ParseInt(string(val), 10, 64)
		return i
	default:
		return 0
	}
}

func toFloat32(v interface{}) float32 {
	switch val := v.(type) {
	case float64:
		return float32(val)
	case string:
		f, _ := strconv.ParseFloat(val, 32)
		return float32(f)
	case []byte:
		f, _ := strconv.ParseFloat(string(val), 32)
		return float32(f)
	default:
		return 0
	}
}

func float32SliceToBytes(vector []float32) []byte {
	bytes := make([]byte, len(vector)*4)
	for i, v := range vector {
		binary.LittleEndian.PutUint32(bytes[i*4:], math.Float32bits(v))
	}
	return bytes
}
