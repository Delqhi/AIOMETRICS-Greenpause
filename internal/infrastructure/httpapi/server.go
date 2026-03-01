package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"greenpause/internal/application"
	"greenpause/internal/domain"
)

type Server struct {
	scheduleUC *application.ScheduleReminderUseCase
	getUC      *application.GetReminderUseCase
	cancelUC   *application.CancelReminderUseCase
	ackUC      *application.AcknowledgeReminderUseCase

	mux *http.ServeMux
}

func NewServer(
	scheduleUC *application.ScheduleReminderUseCase,
	getUC *application.GetReminderUseCase,
	cancelUC *application.CancelReminderUseCase,
	ackUC *application.AcknowledgeReminderUseCase,
) (*Server, error) {
	if scheduleUC == nil || getUC == nil || cancelUC == nil || ackUC == nil {
		return nil, errors.New("http server dependencies must not be nil")
	}

	s := &Server{
		scheduleUC: scheduleUC,
		getUC:      getUC,
		cancelUC:   cancelUC,
		ackUC:      ackUC,
		mux:        http.NewServeMux(),
	}
	s.registerRoutes()
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/v1/reminders", s.handleRemindersRoot)
	s.mux.HandleFunc("/v1/reminders/", s.handleReminderByID)
	s.mux.HandleFunc("/healthz", s.handleHealth)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRemindersRoot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed", correlationIDFromRequest(r))
		return
	}

	var req createReminderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidRequest", err.Error(), correlationIDFromRequest(r))
		return
	}

	if strings.TrimSpace(req.IdempotencyKey) == "" {
		writeError(w, http.StatusBadRequest, "InvalidRequest", "IdempotencyKey is required", correlationIDFromRequest(r))
		return
	}

	result, err := s.scheduleUC.Execute(r.Context(), application.ScheduleReminderCommand{
		TenantID:       domain.TenantID(req.TenantID),
		UserID:         domain.UserID(req.UserID),
		DueAtUtc:       req.DueAtUtc,
		Message:        req.Message,
		IdempotencyKey: req.IdempotencyKey,
		CorrelationID:  correlationIDFromRequest(r),
	})
	if err != nil {
		writeMappedError(w, err, correlationIDFromRequest(r))
		return
	}

	view, err := s.getUC.Execute(r.Context(), application.GetReminderQuery{
		TenantID:   domain.TenantID(req.TenantID),
		ReminderID: result.ReminderID,
	})
	if err != nil {
		writeMappedError(w, err, correlationIDFromRequest(r))
		return
	}

	writeJSON(w, http.StatusCreated, createReminderResponse{
		ReminderID: string(result.ReminderID),
		Reminder:   reminderItemFromView(view),
	})
}

func (s *Server) handleReminderByID(w http.ResponseWriter, r *http.Request) {
	reminderID, ok := reminderIDFromPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "NotFound", "resource not found", correlationIDFromRequest(r))
		return
	}
	tenantID := tenantIDFromRequest(r)
	if strings.TrimSpace(tenantID) == "" {
		writeError(w, http.StatusBadRequest, "InvalidRequest", "TenantId is required via query parameter TenantId or header X-Tenant-Id", correlationIDFromRequest(r))
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetReminder(w, r, tenantID, reminderID)
	case http.MethodPatch:
		s.handlePatchReminder(w, r, tenantID, reminderID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "MethodNotAllowed", "method not allowed", correlationIDFromRequest(r))
	}
}

func (s *Server) handleGetReminder(w http.ResponseWriter, r *http.Request, tenantID string, reminderID string) {
	view, err := s.getUC.Execute(r.Context(), application.GetReminderQuery{
		TenantID:   domain.TenantID(tenantID),
		ReminderID: domain.ReminderID(reminderID),
	})
	if err != nil {
		writeMappedError(w, err, correlationIDFromRequest(r))
		return
	}
	writeJSON(w, http.StatusOK, getReminderResponse{Reminder: reminderItemFromView(view)})
}

func (s *Server) handlePatchReminder(w http.ResponseWriter, r *http.Request, tenantID string, reminderID string) {
	var req updateReminderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "InvalidRequest", err.Error(), correlationIDFromRequest(r))
		return
	}
	if strings.TrimSpace(req.Action) == "" {
		writeError(w, http.StatusBadRequest, "InvalidRequest", "Action is required", correlationIDFromRequest(r))
		return
	}

	action := strings.ToLower(strings.TrimSpace(req.Action))
	switch action {
	case "cancel":
		if err := s.cancelUC.Execute(r.Context(), application.CancelReminderCommand{
			TenantID:      domain.TenantID(tenantID),
			ReminderID:    domain.ReminderID(reminderID),
			CorrelationID: correlationIDFromRequest(r),
		}); err != nil {
			writeMappedError(w, err, correlationIDFromRequest(r))
			return
		}
	case "acknowledge":
		if err := s.ackUC.Execute(r.Context(), application.AcknowledgeReminderCommand{
			TenantID:      domain.TenantID(tenantID),
			ReminderID:    domain.ReminderID(reminderID),
			CorrelationID: correlationIDFromRequest(r),
		}); err != nil {
			writeMappedError(w, err, correlationIDFromRequest(r))
			return
		}
	default:
		writeError(w, http.StatusBadRequest, "UnsupportedAction", "supported actions: Cancel, Acknowledge", correlationIDFromRequest(r))
		return
	}

	view, err := s.getUC.Execute(r.Context(), application.GetReminderQuery{
		TenantID:   domain.TenantID(tenantID),
		ReminderID: domain.ReminderID(reminderID),
	})
	if err != nil {
		writeMappedError(w, err, correlationIDFromRequest(r))
		return
	}

	writeJSON(w, http.StatusOK, reminderItemFromView(view))
}

type createReminderRequest struct {
	TenantID       string    `json:"TenantId"`
	UserID         string    `json:"UserId"`
	DueAtUtc       time.Time `json:"DueAtUtc"`
	Message        string    `json:"Message"`
	Channel        string    `json:"Channel,omitempty"`
	IdempotencyKey string    `json:"IdempotencyKey"`
}

type createReminderResponse struct {
	ReminderID string       `json:"ReminderId"`
	Reminder   reminderItem `json:"Reminder"`
}

type getReminderResponse struct {
	Reminder reminderItem `json:"Reminder"`
}

type updateReminderRequest struct {
	Action string `json:"Action"`
}

type reminderItem struct {
	ReminderID     string     `json:"ReminderId"`
	TenantID       string     `json:"TenantId"`
	UserID         string     `json:"UserId"`
	DueAtUtc       time.Time  `json:"DueAtUtc"`
	ReminderStatus string     `json:"ReminderStatus"`
	Message        string     `json:"Message"`
	CreatedAtUtc   time.Time  `json:"CreatedAtUtc"`
	UpdatedAtUtc   time.Time  `json:"UpdatedAtUtc"`
	TriggeredAtUtc *time.Time `json:"TriggeredAtUtc,omitempty"`
	CanceledAtUtc  *time.Time `json:"CanceledAtUtc,omitempty"`
	AcknowledgedAt *time.Time `json:"AcknowledgedAtUtc,omitempty"`
}

type errorResponse struct {
	ErrorCode     string `json:"ErrorCode"`
	ErrorMessage  string `json:"ErrorMessage"`
	CorrelationID string `json:"CorrelationId"`
}

func reminderItemFromView(view application.ReminderView) reminderItem {
	updatedAt := view.CreatedAtUtc
	if view.TriggeredAtUtc != nil && view.TriggeredAtUtc.After(updatedAt) {
		updatedAt = *view.TriggeredAtUtc
	}
	if view.CanceledAtUtc != nil && view.CanceledAtUtc.After(updatedAt) {
		updatedAt = *view.CanceledAtUtc
	}
	if view.AcknowledgedAt != nil && view.AcknowledgedAt.After(updatedAt) {
		updatedAt = *view.AcknowledgedAt
	}

	return reminderItem{
		ReminderID:     string(view.ReminderID),
		TenantID:       string(view.TenantID),
		UserID:         string(view.UserID),
		DueAtUtc:       view.DueAtUtc,
		ReminderStatus: string(view.ReminderStatus),
		Message:        view.Message,
		CreatedAtUtc:   view.CreatedAtUtc,
		UpdatedAtUtc:   updatedAt,
		TriggeredAtUtc: view.TriggeredAtUtc,
		CanceledAtUtc:  view.CanceledAtUtc,
		AcknowledgedAt: view.AcknowledgedAt,
	}
}

func writeMappedError(w http.ResponseWriter, err error, correlationID string) {
	switch {
	case errors.Is(err, application.ErrReminderNotFound):
		writeError(w, http.StatusNotFound, "ReminderNotFound", err.Error(), correlationID)
	case errors.Is(err, domain.ErrInvalidDueAt),
		errors.Is(err, domain.ErrInvalidReminderID),
		errors.Is(err, domain.ErrInvalidTenantID),
		errors.Is(err, domain.ErrInvalidUserID),
		errors.Is(err, domain.ErrInvalidMessage),
		errors.Is(err, domain.ErrInvalidIdempotencyKey):
		writeError(w, http.StatusBadRequest, "ValidationError", err.Error(), correlationID)
	case errors.Is(err, domain.ErrInvalidStatusTransition):
		writeError(w, http.StatusConflict, "InvalidTransition", err.Error(), correlationID)
	default:
		writeError(w, http.StatusInternalServerError, "InternalError", "internal error", correlationID)
	}
}

func writeError(w http.ResponseWriter, status int, code string, message string, correlationID string) {
	writeJSON(w, status, errorResponse{ErrorCode: code, ErrorMessage: message, CorrelationID: correlationID})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func decodeJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid json payload: %w", err)
	}
	if decoder.More() {
		return fmt.Errorf("invalid json payload: multiple JSON values")
	}
	return nil
}

func reminderIDFromPath(path string) (string, bool) {
	if !strings.HasPrefix(path, "/v1/reminders/") {
		return "", false
	}
	reminderID := strings.TrimPrefix(path, "/v1/reminders/")
	reminderID = strings.TrimSpace(strings.Trim(reminderID, "/"))
	if reminderID == "" || strings.Contains(reminderID, "/") {
		return "", false
	}
	return reminderID, true
}

func tenantIDFromRequest(r *http.Request) string {
	tenantID := strings.TrimSpace(r.URL.Query().Get("TenantId"))
	if tenantID != "" {
		return tenantID
	}
	return strings.TrimSpace(r.Header.Get("X-Tenant-Id"))
}

func correlationIDFromRequest(r *http.Request) string {
	candidate := strings.TrimSpace(r.Header.Get("X-Correlation-Id"))
	if candidate != "" {
		return candidate
	}

	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("corr-%d", time.Now().UTC().UnixNano())
	}
	return "corr-" + hex.EncodeToString(b[:])
}
