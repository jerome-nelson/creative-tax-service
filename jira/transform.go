package main

import (
	"JiraConnect/shared"
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/genai"
	"log"
	"net/http"
	"os"
)

type LLMConfig struct {
	ApiKey string
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

func handlePartiallyGeneratedIssueTransform(log *log.Logger, config LLMConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()

		var payload JSONPayload

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid JSON payload: "+err.Error(), http.StatusBadRequest)
			return
		}

		styleGuidePath := "jira/style-guide.md"
		styleGuideContent, err := os.ReadFile(styleGuidePath)
		if err != nil {
			http.Error(w, "failed to read style guide", http.StatusInternalServerError)
			log.Printf("error reading style guide: %v", err)
			return
		}

		client, err := genai.NewClient(ctx, &genai.ClientConfig{
			APIKey: config.ApiKey,
		})
		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		prompt := fmt.Sprintf(
			"%s\n\nUse the above style guide to transform the following input:\n\nHeading: %s\nDescription: %s\nTask Name: %s",
			string(styleGuideContent),
			payload.Heading,
			payload.Description,
			payload.TaskName,
		)

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

		log.Printf("generating results for prompt")
		rawText, err := client.Models.GenerateContent(
			ctx,
			"gemini-2.0-flash",
			genai.Text(prompt),
			config,
		)

		if err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println(err)
			return
		}

		var result LLMResponse
		if err := json.Unmarshal([]byte(rawText.Text()), &result); err != nil {
			http.Error(w, "failed to parse model output", http.StatusInternalServerError)
			log.Println("JSON parse error:", err)
			return
		}

		if err := shared.Encode(w, http.StatusOK, result); err != nil {
			http.Error(w, "internal server error", http.StatusInternalServerError)
			log.Println(err)
		}
	}
}
