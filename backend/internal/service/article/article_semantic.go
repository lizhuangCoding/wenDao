package article

import (
	"math"
	"sort"

	"wenDao/internal/model"
)

const (
	orbitSemanticNeighborLimit    = 4
	orbitSemanticNeighborMinScore = 0.25
)

type orbitSemanticVector struct {
	articleID int64
	profile   *model.ArticleSemanticProfile
	vector    []float32
}

func (s *articleService) hydrateOrbitSemanticProfiles(articles []*model.Article) error {
	if s.semanticRepo == nil || len(articles) == 0 {
		return nil
	}

	articleIDs := make([]int64, 0, len(articles))
	for _, article := range articles {
		if article == nil {
			continue
		}
		articleIDs = append(articleIDs, article.ID)
	}

	profilesByArticleID, err := s.semanticRepo.ListByArticleIDs(articleIDs)
	if err != nil {
		return err
	}

	vectors := make([]orbitSemanticVector, 0, len(profilesByArticleID))
	for _, article := range articles {
		if article == nil {
			continue
		}
		profile := profilesByArticleID[article.ID]
		if profile == nil {
			continue
		}
		article.SemanticProfile = profile
		vector, err := profile.Embedding()
		if err != nil || len(vector) == 0 {
			continue
		}
		vectors = append(vectors, orbitSemanticVector{
			articleID: article.ID,
			profile:   profile,
			vector:    vector,
		})
	}

	for _, current := range vectors {
		neighbors := topSemanticNeighbors(current, vectors)
		if err := current.profile.SetNeighbors(neighbors); err != nil {
			return err
		}
	}

	return nil
}

func topSemanticNeighbors(current orbitSemanticVector, candidates []orbitSemanticVector) []model.ArticleSemanticNeighbor {
	neighbors := make([]model.ArticleSemanticNeighbor, 0, orbitSemanticNeighborLimit)
	for _, candidate := range candidates {
		if candidate.articleID == current.articleID {
			continue
		}
		score := cosineSimilarity(current.vector, candidate.vector)
		if score < orbitSemanticNeighborMinScore {
			continue
		}
		neighbors = append(neighbors, model.ArticleSemanticNeighbor{
			ArticleID: candidate.articleID,
			Score:     score,
		})
	}
	sort.Slice(neighbors, func(i, j int) bool {
		if neighbors[i].Score == neighbors[j].Score {
			return neighbors[i].ArticleID < neighbors[j].ArticleID
		}
		return neighbors[i].Score > neighbors[j].Score
	})
	if len(neighbors) > orbitSemanticNeighborLimit {
		return neighbors[:orbitSemanticNeighborLimit]
	}
	return neighbors
}

func cosineSimilarity(left, right []float32) float64 {
	if len(left) == 0 || len(left) != len(right) {
		return 0
	}
	var dot, leftNorm, rightNorm float64
	for index, leftValue := range left {
		rightValue := right[index]
		dot += float64(leftValue) * float64(rightValue)
		leftNorm += float64(leftValue) * float64(leftValue)
		rightNorm += float64(rightValue) * float64(rightValue)
	}
	if leftNorm == 0 || rightNorm == 0 {
		return 0
	}
	return dot / (math.Sqrt(leftNorm) * math.Sqrt(rightNorm))
}
