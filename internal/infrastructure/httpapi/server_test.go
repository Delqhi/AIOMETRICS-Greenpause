package httpapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"greenpause/internal/application"
	"greenpause/internal/domain"
	"greenpause/internal/infrastructure/httpapi"
	"greenpause/internal/infrastructure/idgen"
	"greenpause/internal/infrastructure/memory"
)

func TestServer_CreateAndGetReminder(t *testing.T) {
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	repo := memory.NewReminderRepository()
	index := memory.NewScheduleIndex()
	audit := memory.NewAuditLog()
	clock := memory.FixedClock{Current: now}
	idSource := idgen.NewUUIDv7ReminderIDGenerator()

	scheduleUC, err := application.NewScheduleReminderUseCase(repo, index, audit, clock, idSource)
	if err != nil {
		t.Fatalf("new schedule uc: %v", err)
	}
	getUC, err := application.NewGetReminderUseCase(repo)
	if err != nil {
		t.Fatalf("new get uc: %v", err)
	}
	cancelUC, err := application.NewCancelReminderUseCase(repo, index, audit, clock)
	if err != nil {
		t.Fatalf("new cancel uc: %v", err)
	}
	ackUC, err := application.NewAcknowledgeReminderUseCase(repo, audit, clock)
	if err != nil {
		t.Fatalf("new ack uc: %v", err)
	}

	server, err := httpapi.NewServer(scheduleUC, getUC, cancelUC, ackUC)
	if err != nil {
		t.Fatalf("new http server: %v", err)
	}

	createBody := map[string]any{
		"TenantId":       "tenant-a",
		"UserId":         "user-1",
		"DueAtUtc":       now.Add(2 * time.Minute).Format(time.RFC3339),
		"Message":        "Check email",
		"IdempotencyKey": "idem-12345678",
	}
	createPayload, _ := json.Marshal(createBody)

	createReq := httptest.NewRequest(http.MethodPost, "/v1/reminders", bytes.NewReader(createPayload))
	createReq.Header.Set("Content-Type", "application/json")
	createResp := httptest.NewRecorder()
	server.Handler().ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d, body=%s", createResp.Code, createResp.Body.String())
	}

	var created struct {
		ReminderID string `json:"ReminderId"`
	}
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.ReminderID == "" {
		t.Fatalf("expected reminder id")
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/reminders/"+created.ReminderID+"?TenantId=tenant-a", nil)
	getResp := httptest.NewRecorder()
	server.Handler().ServeHTTP(getResp, getReq)
	if getResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", getResp.Code, getResp.Body.String())
	}
}

func TestServer_PatchCancel(t *testing.T) {
	now := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	repo := memory.NewReminderRepository()
	index := memory.NewScheduleIndex()
	audit := memory.NewAuditLog()
	clock := memory.FixedClock{Current: now}
	idSource := idgen.NewUUIDv7ReminderIDGenerator()

	scheduleUC, _ := application.NewScheduleReminderUseCase(repo, index, audit, clock, idSource)
	getUC, _ := application.NewGetReminderUseCase(repo)
	cancelUC, _ := application.NewCancelReminderUseCase(repo, index, audit, clock)
	ackUC, _ := application.NewAcknowledgeReminderUseCase(repo, audit, clock)
	server, _ := httpapi.NewServer(scheduleUC, getUC, cancelUC, ackUC)

	created, err := scheduleUC.Execute(context.Background(), application.ScheduleReminderCommand{
		TenantID:       domain.TenantID("tenant-a"),
		UserID:         domain.UserID("user-1"),
		DueAtUtc:       now.Add(2 * time.Minute),
		Message:        "Check email",
		IdempotencyKey: "idem-12345678",
		CorrelationID:  "corr-1",
	})
	if err != nil {
		t.Fatalf("seed reminder: %v", err)
	}

	patchBody := map[string]any{"Action": "Cancel"}
	payload, _ := json.Marshal(patchBody)
	patchReq := httptest.NewRequest(http.MethodPatch, "/v1/reminders/"+string(created.ReminderID)+"?TenantId=tenant-a", bytes.NewReader(payload))
	patchReq.Header.Set("Content-Type", "application/json")
	patchResp := httptest.NewRecorder()
	server.Handler().ServeHTTP(patchResp, patchReq)
	if patchResp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body=%s", patchResp.Code, patchResp.Body.String())
	}
}
