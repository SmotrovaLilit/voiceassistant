package googlesheetscrm

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"
	"time"

	"golang.org/x/oauth2/google"

	"voiceassistant-api/internal/domain"
)

const (
	sheetsAPIURL      = "https://sheets.googleapis.com/v4/spreadsheets/"
	tasksSheet        = "CRM Tasks"
	visitReportsSheet = "CRM Sales Visit Reports"
	sheetIDKey        = "sheetId"
	cellValueKey      = "userEnteredValue"
)

var headers = map[string][]string{
	tasksSheet:        {"ID", "Client Name", "Title", "Description", "Due At", "Remind At"},
	visitReportsSheet: {"ID", "Client Name", "Contact Name", "Summary", "Visited At"},
}

type Config struct {
	SpreadsheetID   string
	CredentialsFile string
}

type CRMRepository struct {
	endpoint     string
	client       *http.Client
	sheetIDs     map[string]int64
	sheetHeaders map[string][]string
}

func New(ctx context.Context, config Config) (*CRMRepository, error) {
	if strings.TrimSpace(config.SpreadsheetID) == "" {
		return nil, errors.New("GOOGLE_SPREADSHEET_ID is required")
	}
	client, err := newClient(ctx, config)
	if err != nil {
		return nil, err
	}
	crm := &CRMRepository{
		endpoint: sheetsAPIURL + url.PathEscape(config.SpreadsheetID),
		client:   client, sheetIDs: make(map[string]int64),
		sheetHeaders: make(map[string][]string),
	}
	for title, columns := range headers {
		crm.sheetHeaders[title] = slices.Clone(columns)
	}
	if err := crm.ensureSchema(ctx); err != nil {
		return nil, fmt.Errorf("initialize Google Sheets CRM: %w", err)
	}
	return crm, nil
}

func newClient(ctx context.Context, config Config) (*http.Client, error) {
	if strings.TrimSpace(config.CredentialsFile) == "" {
		return nil, errors.New("GOOGLE_CREDENTIALS_FILE is required")
	}
	credentials, err := os.ReadFile(config.CredentialsFile)
	if err != nil {
		return nil, fmt.Errorf("read Google credentials: %w", err)
	}
	jwtConfig, err := google.JWTConfigFromJSON(credentials, "https://www.googleapis.com/auth/spreadsheets")
	if err != nil {
		return nil, fmt.Errorf("parse Google credentials: %w", err)
	}
	client := jwtConfig.Client(ctx)
	client.Timeout = 30 * time.Second
	return client, nil
}

func (g *CRMRepository) SaveWorkflows(ctx context.Context, workflows []domain.Workflow) error {
	requests := make([]any, 0, len(workflows))
	for _, workflow := range workflows {
		validated, err := domain.NewWorkflow(workflow.Payload)
		if err != nil {
			return fmt.Errorf("validate CRM workflow: %w", err)
		}
		if validated.Type != workflow.Type {
			return errors.New("workflow type does not match its payload")
		}
		values := map[string]string{"ID": rand.Text()}
		var sheet string
		switch payload := workflow.Payload.(type) {
		case domain.TaskPayload:
			sheet = tasksSheet
			values["Client Name"], values["Title"] = payload.ClientName, payload.Title
			values["Description"] = payload.Description
			values["Due At"], values["Remind At"] = payload.DueAt, payload.RemindAt
		case domain.VisitReportPayload:
			sheet = visitReportsSheet
			values["Client Name"], values["Contact Name"] = payload.ClientName, payload.ContactName
			values["Summary"], values["Visited At"] = payload.Summary, payload.VisitedAt
		default:
			return fmt.Errorf("unsupported CRM workflow payload %T", workflow.Payload)
		}
		row := make([]string, len(g.sheetHeaders[sheet]))
		for index, column := range g.sheetHeaders[sheet] {
			row[index] = values[column]
		}
		requests = append(requests, map[string]any{"appendCells": map[string]any{
			sheetIDKey: g.sheetIDs[sheet], "rows": []any{cellRow(row)}, "fields": cellValueKey,
		}})
	}
	if len(requests) == 0 {
		return nil
	}
	return g.batchUpdate(ctx, requests)
}

func (g *CRMRepository) ensureSchema(ctx context.Context) error {
	var metadata struct {
		Sheets []struct {
			Properties struct {
				SheetID int64  `json:"sheetId"`
				Title   string `json:"title"`
			} `json:"properties"`
		} `json:"sheets"`
	}
	if err := g.request(ctx, http.MethodGet, "?fields=sheets.properties(sheetId,title)", nil, &metadata); err != nil {
		return err
	}
	var nextID int64
	for _, sheet := range metadata.Sheets {
		g.sheetIDs[sheet.Properties.Title] = sheet.Properties.SheetID
		nextID = max(nextID, sheet.Properties.SheetID+1)
	}
	var requests []any
	for _, title := range []string{tasksSheet, visitReportsSheet} {
		id, exists := g.sheetIDs[title]
		if exists {
			matches, err := g.checkHeader(ctx, title)
			if err != nil {
				return err
			}
			if matches {
				continue
			}
		} else {
			id, nextID = nextID, nextID+1
			g.sheetIDs[title] = id
			requests = append(requests, map[string]any{"addSheet": map[string]any{
				"properties": map[string]any{sheetIDKey: id, "title": title},
			}})
		}
		requests = append(requests, map[string]any{"updateCells": map[string]any{
			"start": map[string]any{sheetIDKey: id, "rowIndex": 0, "columnIndex": 0},
			"rows":  []any{cellRow(headers[title])}, "fields": cellValueKey,
		}})
	}
	if len(requests) == 0 {
		return nil
	}
	return g.batchUpdate(ctx, requests)
}

func (g *CRMRepository) checkHeader(ctx context.Context, title string) (bool, error) {
	var response struct {
		Values [][]string `json:"values"`
	}
	suffix := "/values/" + url.PathEscape("'"+title+"'!1:1")
	if err := g.request(ctx, http.MethodGet, suffix, nil, &response); err != nil {
		return false, err
	}
	if len(response.Values) == 0 || len(response.Values[0]) == 0 {
		return false, nil
	}
	for _, column := range headers[title] {
		if !slices.Contains(response.Values[0], column) {
			return false, fmt.Errorf("sheet %q must have column %q", title, column)
		}
	}
	g.sheetHeaders[title] = response.Values[0]
	return true, nil
}

func cellRow(values []string) map[string]any {
	cells := make([]any, len(values))
	for index, value := range values {
		cells[index] = map[string]any{cellValueKey: map[string]string{"stringValue": value}}
	}
	return map[string]any{"values": cells}
}

func (g *CRMRepository) batchUpdate(ctx context.Context, requests []any) error {
	return g.request(ctx, http.MethodPost, ":batchUpdate", map[string]any{"requests": requests}, nil)
}

func (g *CRMRepository) request(ctx context.Context, method, suffix string, body, destination any) error {
	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode Google Sheets request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}
	request, err := http.NewRequestWithContext(ctx, method, g.endpoint+suffix, reader)
	if err != nil {
		return fmt.Errorf("create Google Sheets request: %w", err)
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := g.client.Do(request)
	if err != nil {
		return fmt.Errorf("call Google Sheets: %w", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("request to Google Sheets returned HTTP %d", response.StatusCode)
	}
	if destination != nil {
		if err := json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(destination); err != nil {
			return fmt.Errorf("decode Google Sheets response: %w", err)
		}
	}
	return nil
}

var _ domain.CRMRepository = (*CRMRepository)(nil)
