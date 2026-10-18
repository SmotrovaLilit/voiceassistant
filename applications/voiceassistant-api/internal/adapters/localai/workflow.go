package localai

import (
	"encoding/json"
	"errors"
	"fmt"

	"voiceassistant-api/internal/domain"
)

const jsonTypeKey = "type"

func decodeWorkflows(text string) ([]domain.Workflow, error) {
	var extracted struct {
		Tasks        []domain.TaskPayload        `json:"tasks"`
		VisitReports []domain.VisitReportPayload `json:"visitreports"`
	}
	if err := json.Unmarshal([]byte(text), &extracted); err != nil {
		return nil, fmt.Errorf("decode LocalAI workflows: %w", err)
	}
	if extracted.Tasks == nil || extracted.VisitReports == nil {
		return nil, errors.New("LocalAI must return tasks and visitreports arrays")
	}
	payloads := make([]domain.WorkflowPayload, 0, len(extracted.Tasks)+len(extracted.VisitReports))
	for _, report := range extracted.VisitReports {
		payloads = append(payloads, report)
	}
	for _, task := range extracted.Tasks {
		payloads = append(payloads, task)
	}
	workflows := make([]domain.Workflow, 0, len(payloads))
	for _, payload := range payloads {
		workflow, err := domain.NewWorkflow(payload)
		if err != nil {
			return nil, fmt.Errorf("invalid %s workflow: %w", payload.Type(), err)
		}
		workflows = append(workflows, workflow)
	}
	return workflows, nil
}

func workflowSchema() map[string]any {
	collection := func(fields ...string) map[string]any {
		properties := make(map[string]any, len(fields))
		for _, field := range fields {
			properties[field] = map[string]string{jsonTypeKey: "string"}
		}
		return map[string]any{
			jsonTypeKey: "array",
			"items": map[string]any{
				jsonTypeKey: "object", "properties": properties,
				"required": fields, "additionalProperties": false,
			},
		}
	}
	return map[string]any{
		jsonTypeKey: "object",
		"properties": map[string]any{
			"tasks":        collection("client_name", "title", "description", "due_at", "remind_at"),
			"visitreports": collection("client_name", "contact_name", "summary", "visited_at"),
		},
		"required": []string{"tasks", "visitreports"}, "additionalProperties": false,
	}
}
