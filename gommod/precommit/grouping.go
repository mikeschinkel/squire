package precommit

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"strings"
	"text/template"

	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/gomion/gommod/askai"
	"github.com/mikeschinkel/gomion/gommod/gitutils"
)

// GroupingArgs contains arguments for commit grouping suggestions
type GroupingArgs struct {
	StagedFiles []dt.RelFilepath
	Analysis    Results
	AIAgent     *askai.Agent
}

// SuggestGroupings asks AI to suggest logical groupings for staged changes
func SuggestGroupings(ctx context.Context, args GroupingArgs) (groups []CommitGroup, err error) {
	var prompt string
	var response string
	var rawGroups []struct {
		Title     string   `json:"title"`
		Files     []string `json:"files"`
		Rationale string   `json:"rationale"`
	}

	// Build AI prompt
	prompt, err = buildGroupingPrompt(args)
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "build_prompt", err)
		goto end
	}

	// Ask AI for grouping suggestions
	response, err = args.AIAgent.Ask(ctx, prompt)
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "ask_ai", err)
		goto end
	}

	// Parse JSON response
	err = json.Unmarshal([]byte(response), &rawGroups)
	if err != nil {
		err = NewErr(ErrPrecommit, "operation", "parse_json", err)
		goto end
	}

	// Convert to CommitGroup structs
	groups = make([]CommitGroup, len(rawGroups))
	for i, rg := range rawGroups {
		groups[i] = CommitGroup{
			Title:     rg.Title,
			Files:     convertToRelFilepaths(rg.Files),
			Rationale: rg.Rationale,
			Suggested: true,
		}
	}

end:
	return groups, err
}

const (
	classifyCommitCategoriesTemplate       = "classify-commit-categorizes.gotmpl"
	generatecommitMessageCandidateTemplate = "generate-commit-message-candidate.gotmpl"
)

// templFS is an embedded filesystem thal allows acceess to the compiled-in templates.
//
//go:embed templates
var templFS embed.FS

// buildGroupingPrompt constructs an AI prompt for commit grouping suggestions
func buildGroupingPrompt(args GroupingArgs) (prompt string, err error) {
	var buf bytes.Buffer
	var tmplText []byte
	var tmpl *template.Template
	var data struct {
		StagedFiles      []dt.RelFilepath
		AnalysisMarkdown string
	}

	// TODO: Add functionality to save to then loaded from the config directory so
	//  they can be modified by user.
	tmplText, err = templFS.ReadFile(classifyCommitCategoriesTemplate)
	if err != nil {
		goto end
	}
	tmpl, err = template.New("grouping").Parse(string(tmplText))
	if err != nil {
		goto end
	}

	data.StagedFiles = args.StagedFiles
	data.AnalysisMarkdown = args.Analysis.FormatForAI()

	err = tmpl.Execute(&buf, data)
	if err != nil {
		goto end
	}

	prompt = buf.String()

end:
	return prompt, err
}

// InteractiveRestage guides user through restaging files according to groups
func InteractiveRestage(ctx context.Context, groups []CommitGroup, repo *gitutils.Repo) (err error) {
	// TODO: Implement interactive restaging
	// For each group:
	//   1. Show files in group
	//   2. Ask user: [a]ccept, [m]odify, [s]kip
	//   3. If accept: unstage all, restage this group, commit
	//   4. If modify: interactive file selection
	//   5. If skip: move to next group
	// After all groups, show remaining unstaged files

	err = NewErr(ErrPrecommit, "operation", "not_implemented",
		"message", "InteractiveRestage is not yet implemented")
	return err
}

// convertToRelFilepaths converts string slices to RelFilepath slices
func convertToRelFilepaths(paths []string) []dt.RelFilepath {
	result := make([]dt.RelFilepath, len(paths))
	for i, path := range paths {
		result[i] = dt.RelFilepath(strings.TrimSpace(path))
	}
	return result
}
