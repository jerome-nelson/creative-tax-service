package main

import (
	"JiraConnect/shared"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"google.golang.org/genai"
	"log"
	"net/http"
	"os"
)

type LLMConfig struct {
	ApiKey     string
	KiroURL    string
	UseKiro    bool
}

type LLMResponse struct {
	Heading     string   `json:"heading"`
	Description string   `json:"description"`
	Links       []string `json:"links"`
}

type JSONPayload struct {
	Heading     string   `json:"heading"`
	Description []string `json:"description"`
	TaskName    string   `json:"taskName"`
}

type KiroRequest struct {
	Prompt string `json:"prompt"`
}

type KiroResponse struct {
	Response string `json:"response"`
}

func transformWithKiro(ctx context.Context, prompt string, kiroURL string, log *log.Logger) (*LLMResponse, error) {
	// Create request to Kiro
	reqBody := KiroRequest{Prompt: prompt}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", kiroURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Kiro API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Kiro API returned status %d", resp.StatusCode)
	}

	var kiroResp KiroResponse
	if err := json.NewDecoder(resp.Body).Decode(&kiroResp); err != nil {
		return nil, fmt.Errorf("failed to decode Kiro response: %w", err)
	}

	// Parse the JSON response from Kiro
	var result LLMResponse
	if err := json.Unmarshal([]byte(kiroResp.Response), &result); err != nil {
		// If parsing fails, return a basic response
		log.Printf("Failed to parse Kiro response as JSON, using raw response")
		return &LLMResponse{
			Heading:     "Transformed via Kiro",
			Description: kiroResp.Response,
			Links:       []string{},
		}, nil
	}

	return &result, nil
}

func transformWithGemini(ctx context.Context, prompt string, apiKey string, log *log.Logger) (*LLMResponse, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey: apiKey,
		HTTPClient: &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // TODO: Fix certificate chain issue
				},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	config := &genai.GenerateContentConfig{
		ResponseMIMEType: "application/json",
		ResponseSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"heading":     {Type: genai.TypeString},
				"description": {Type: genai.TypeString},
				"links": {
					Type:  genai.TypeArray,
					Items: &genai.Schema{Type: genai.TypeString},
				},
			},
			PropertyOrdering: []string{"heading", "description", "links"},
		},
	}

	log.Printf("generating results with Gemini")
	rawText, err := client.Models.GenerateContent(
		ctx,
		"gemini-2.0-flash",
		genai.Text(prompt),
		config,
	)

	if err != nil {
		return nil, fmt.Errorf("Gemini API error: %w", err)
	}

	var result LLMResponse
	if err := json.Unmarshal([]byte(rawText.Text()), &result); err != nil {
		return nil, fmt.Errorf("failed to parse Gemini output: %w", err)
	}

	return &result, nil
}

func handlePartiallyGeneratedIssueTransform(log *log.Logger, config LLMConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		var payload JSONPayload

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
			return
		}

		styleGuidePath := "jira/templates/style-guide.md"
		styleGuideContent, err := os.ReadFile(styleGuidePath)
		if err != nil {
			http.Error(w, "failed to read style guide", http.StatusInternalServerError)
			log.Printf("error reading style guide: %v", err)
			return
		}

		prompt := fmt.Sprintf(
			"%s\n\nUse the above style guide to transform the following input:\n\nHeading: %s\nDescription: %s\nTask Name: %s\n\nProvide the output in JSON format with keys: heading, description, links (array)",
			string(styleGuideContent),
			payload.Heading,
			payload.Description,
			payload.TaskName,
		)

		var result *LLMResponse
		
		if config.UseKiro {
			log.Printf("Using Kiro for transformation")
			result, err = transformWithKiro(ctx, prompt, config.KiroURL, log)
		} else {
			log.Printf("Using Gemini for transformation")
			result, err = transformWithGemini(ctx, prompt, config.ApiKey, log)
		}

		if err != nil {
			http.Error(w, "transformation failed: "+err.Error(), http.StatusInternalServerError)
			log.Printf("transformation error: %v", err)
			return
		}

		if err := shared.Encode(w, http.StatusOK, result); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println(err)
		}
	}
}
