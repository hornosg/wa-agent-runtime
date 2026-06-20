package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// VoyageEmbeddings — adaptador real (voyage-3.5, dim 1024 — D-02).
// API: POST https://api.voyageai.com/v1/embeddings, Bearer VOYAGE_API_KEY.
type VoyageEmbeddings struct {
	apiKey string
	model  string
	http   *http.Client
}

func NewVoyageEmbeddings(apiKey string) *VoyageEmbeddings {
	return &VoyageEmbeddings{
		apiKey: apiKey,
		model:  "voyage-3.5",
		http:   &http.Client{Timeout: 30 * time.Second},
	}
}

type voyageReq struct {
	Input     []string `json:"input"`
	Model     string   `json:"model"`
	InputType string   `json:"input_type,omitempty"`
}

type voyageResp struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
}

func (v *VoyageEmbeddings) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	body, _ := json.Marshal(voyageReq{Input: texts, Model: v.model, InputType: "document"})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.voyageai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+v.apiKey)

	resp, err := v.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("voyage %d: %s", resp.StatusCode, string(b))
	}

	var out voyageResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	vecs := make([][]float32, len(out.Data))
	for i, d := range out.Data {
		vecs[i] = d.Embedding
	}
	return vecs, nil
}
