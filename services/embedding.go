package services

import (
	"context"
	"department/label_scanner/config"
	"encoding/json"
	"log"
	"math"
	"os"

	"github.com/sashabaranov/go-openai"
)

var words = []string{
	"type",
	"make",
	"model",
	"cpu_model",
	"cpu_series",
	"serial_number",
	"part_number",
	"battery",
	"adapter",
	"ram_capacity_size",
	"screen_size_inches",
	"hdd_capacity",
	"hdd_type",
	"ram_type",
	"cpu_speed",
	"gpu_model",
	"cam",
}

type EmbeddingService struct {
	vectorsData  map[string][]float32
	OpenAiClient *openai.Client
}

func NewEmbeddingService() *EmbeddingService {
	vectorsData := LoadVectors()
	openAiClient := openai.NewClient(config.GetConfig().EmbeddingConfig.OpenaiApiKey)
	return &EmbeddingService{
		vectorsData:  vectorsData,
		OpenAiClient: openAiClient,
	}
}

func (p *EmbeddingService) FindRelatedType(word string) string {
	queryEmbedding, err := p.GetEmbedding(word)
	if err != nil {
		return ""
	}

	maxRelatedness := float32(0.0)
	relatedWord := ""
	for _, word := range words {
		embedding, err := p.GetEmbedding(word)
		if err != nil {
			continue
		}

		relatedness := p.GetRelatedness(queryEmbedding, embedding)
		if relatedness > maxRelatedness {
			maxRelatedness = relatedness
			relatedWord = word
		}
	}

	if maxRelatedness < 0.7 {
		return ""
	}

	return relatedWord
}

func (p *EmbeddingService) GetEmbedding(text string) ([]float32, error) {
	if value, ok := p.vectorsData[text]; ok {
		return value, nil
	}

	embedding, err := p.OpenAiClient.CreateEmbeddings(context.Background(), openai.EmbeddingRequest{
		Model:      openai.LargeEmbedding3,
		Input:      text,
		Dimensions: 999,
	})

	if err != nil {
		return nil, err
	}

	embed := embedding.Data[0].Embedding
	p.vectorsData[text] = embed
	jsonData, err := json.Marshal(p.vectorsData)
	if err != nil {
		return nil, err
	}

	os.WriteFile(config.GetConfig().EmbeddingConfig.VectorsFile, jsonData, 0644)
	return embed, nil
}

func LoadVectors() map[string][]float32 {
	file, err := os.ReadFile(config.GetConfig().EmbeddingConfig.VectorsFile)
	if err != nil {
		log.Fatalf("Failed to read vectors file: %v", err)
	}

	var vectorsData map[string][]float32
	err = json.Unmarshal(file, &vectorsData)
	if err != nil {
		log.Fatalf("Failed to unmarshal vectors: %v", err)
	}

	return vectorsData
}

func (p *EmbeddingService) GetRelatedness(queryEmbedding []float32, dfEmbedding []float32) float32 {
	// Calculate dot product
	var dotProduct float32
	for i := range queryEmbedding {
		dotProduct += queryEmbedding[i] * dfEmbedding[i]
	}

	// Calculate magnitudes
	var queryMagnitude float32
	var dfMagnitude float32

	for i := range queryEmbedding {
		queryMagnitude += queryEmbedding[i] * queryEmbedding[i]
		dfMagnitude += dfEmbedding[i] * dfEmbedding[i]
	}

	queryMagnitude = float32(math.Sqrt(float64(queryMagnitude)))
	dfMagnitude = float32(math.Sqrt(float64(dfMagnitude)))

	// Calculate cosine similarity
	cosineSimilarity := dotProduct / (queryMagnitude * dfMagnitude)

	// The original Python function returns 1 - cosine distance
	// Since cosine similarity is already what we want, we return it directly
	return cosineSimilarity
}
