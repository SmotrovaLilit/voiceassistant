package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type WorkflowType string

const (
	TaskWorkflowType        WorkflowType = "task"
	VisitReportWorkflowType WorkflowType = "visitreport"
)

type WorkflowPayload interface {
	Type() WorkflowType
}

type Workflow struct {
	Type    WorkflowType    `json:"type"`
	Payload WorkflowPayload `json:"payload"`
}

type TaskPayload struct {
	ClientName  string `json:"client_name,omitempty"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	DueAt       string `json:"due_at,omitempty"`
	RemindAt    string `json:"remind_at,omitempty"`
}

func (TaskPayload) Type() WorkflowType { return TaskWorkflowType }

type VisitReportPayload struct {
	ClientName  string `json:"client_name,omitempty"`
	ContactName string `json:"contact_name,omitempty"`
	Summary     string `json:"summary"`
	VisitedAt   string `json:"visited_at,omitempty"`
}

func (VisitReportPayload) Type() WorkflowType { return VisitReportWorkflowType }

func NewWorkflow(payload WorkflowPayload) (Workflow, error) {
	var dates []string
	switch value := payload.(type) {
	case TaskPayload:
		if strings.TrimSpace(value.Title) == "" {
			return Workflow{}, errors.New("task title is required")
		}
		dates = []string{value.DueAt, value.RemindAt}
	case VisitReportPayload:
		if strings.TrimSpace(value.Summary) == "" {
			return Workflow{}, errors.New("visit report summary is required")
		}
		dates = []string{value.VisitedAt}
	default:
		return Workflow{}, errors.New("unsupported workflow payload")
	}
	for _, date := range dates {
		if date == "" {
			continue
		}
		if _, err := time.Parse(time.DateOnly, date); err == nil {
			continue
		}
		if _, err := time.Parse(time.RFC3339, date); err != nil {
			return Workflow{}, fmt.Errorf("workflow date must use YYYY-MM-DD or RFC3339: %w", err)
		}
	}
	return Workflow{Type: payload.Type(), Payload: payload}, nil
}
