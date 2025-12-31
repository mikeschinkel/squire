package gompkg

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"text/template"
	"time"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/askai"
)

//go:embed plan_takes_prompt.tmpl
var planTakesPromptTemplate string

// GeneratePlanTakes calls AI to generate 3 different takes on how to group changes
func GeneratePlanTakes(
	ctx context.Context,
	agent *askai.Agent,
	files []dt.RelFilepath,
	diff string,
) (takes *PlanTakes, err error) {
	var prompt string
	var response string
	var cfgTakes AITakesResponse

	// Build prompt from template
	prompt, err = buildPlanTakesPrompt(files, diff)
	if err != nil {
		goto end
	}

	// Call AI
	response, err = agent.Ask(ctx, prompt)
	if err != nil {
		goto end
	}

	// Parse JSON response
	err = json.Unmarshal([]byte(response), &cfgTakes)
	if err != nil {
		goto end
	}

	// Convert to validated types
	takes, err = convertAIResponseToPlanTakes(cfgTakes, files)
	if err != nil {
		goto end
	}

end:
	return takes, err
}

// AITakesResponse matches the JSON structure from the AI
type AITakesResponse struct {
	Takes []AITake `json:"takes"`
}

type AITake struct {
	Number int           `json:"number"`
	Theme  string        `json:"theme"`
	Groups []AIChangeSet `json:"groups"`
}

type AIChangeSet struct {
	Name      string   `json:"name"`
	Rationale string   `json:"rationale"`
	Files     []string `json:"files"`
}

// buildPlanTakesPrompt renders the template with file list and diff
func buildPlanTakesPrompt(files []dt.RelFilepath, diff string) (prompt string, err error) {
	var tmpl *template.Template
	var buf bytes.Buffer

	// Convert files to strings for template
	fileStrs := make([]string, len(files))
	for i, f := range files {
		fileStrs[i] = string(f)
	}

	// Parse template
	tmpl, err = template.New("plan_takes").Parse(planTakesPromptTemplate)
	if err != nil {
		goto end
	}

	// Execute template
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Files": fileStrs,
		"Diff":  diff,
	})
	if err != nil {
		goto end
	}

	prompt = buf.String()

end:
	return prompt, err
}

// convertAIResponseToPlanTakes converts AI JSON response to validated PlanTakes
func convertAIResponseToPlanTakes(
	aiResponse AITakesResponse,
	allFiles []dt.RelFilepath,
) (takes *PlanTakes, err error) {
	takes = &PlanTakes{
		Timestamp: time.Now(),
		Takes:     make([]PlanTake, len(aiResponse.Takes)),
	}

	for i, aiTake := range aiResponse.Takes {
		takes.Takes[i] = PlanTake{
			Number:    aiTake.Number,
			Theme:     aiTake.Theme,
			ChangeSet: make([]ChangeSet, len(aiTake.Groups)),
		}

		for j, aiGroup := range aiTake.Groups {
			files := make([]dt.RelFilepath, len(aiGroup.Files))
			for k, f := range aiGroup.Files {
				files[k] = dt.RelFilepath(f)
			}

			takes.Takes[i].ChangeSet[j] = ChangeSet{
				Name:      aiGroup.Name,
				Rationale: aiGroup.Rationale,
				Files:     files,
			}
		}
	}

	return takes, nil
}

// CreateDefaultTake creates a simple "stage everything" take
func CreateDefaultTake(files []dt.RelFilepath) *PlanTakes {
	return &PlanTakes{
		Timestamp: time.Now(),
		Takes: []PlanTake{
			{
				Number: 0,
				Theme:  "All Changes",
				ChangeSet: []ChangeSet{
					{
						Name:      "Stage everything",
						Rationale: "Commit all changes together",
						Files:     files,
					},
				},
			},
		},
	}
}
