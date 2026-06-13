package model

import (
	"encoding/json"
	"time"
)

type ArticleSemanticNeighbor struct {
	ArticleID int64   `json:"article_id"`
	Score     float64 `json:"score"`
}

// ArticleSemanticProfile stores article-level semantics for visual knowledge maps.
// RAG continues to use chunk vectors; this profile is the article-level aggregate.
type ArticleSemanticProfile struct {
	ArticleID     int64     `gorm:"primaryKey;autoIncrement:false" json:"article_id"`
	EmbeddingJSON string    `gorm:"type:longtext;not null" json:"-"`
	ContentHash   string    `gorm:"size:64;not null;index" json:"content_hash"`
	MapX          float64   `gorm:"not null;default:0" json:"map_x"`
	MapY          float64   `gorm:"not null;default:0" json:"map_y"`
	MapZ          float64   `gorm:"not null;default:0" json:"map_z"`
	NeighborJSON  string    `gorm:"type:mediumtext" json:"-"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (ArticleSemanticProfile) TableName() string {
	return "article_semantic_profiles"
}

func (p *ArticleSemanticProfile) SetEmbedding(vector []float32) error {
	data, err := json.Marshal(vector)
	if err != nil {
		return err
	}
	p.EmbeddingJSON = string(data)
	return nil
}

func (p *ArticleSemanticProfile) Embedding() ([]float32, error) {
	if p == nil || p.EmbeddingJSON == "" {
		return nil, nil
	}
	var vector []float32
	if err := json.Unmarshal([]byte(p.EmbeddingJSON), &vector); err != nil {
		return nil, err
	}
	return vector, nil
}

func (p *ArticleSemanticProfile) SetNeighbors(neighbors []ArticleSemanticNeighbor) error {
	data, err := json.Marshal(neighbors)
	if err != nil {
		return err
	}
	p.NeighborJSON = string(data)
	return nil
}

func (p *ArticleSemanticProfile) Neighbors() ([]ArticleSemanticNeighbor, error) {
	if p == nil || p.NeighborJSON == "" {
		return nil, nil
	}
	var neighbors []ArticleSemanticNeighbor
	if err := json.Unmarshal([]byte(p.NeighborJSON), &neighbors); err != nil {
		return nil, err
	}
	return neighbors, nil
}
