package api

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/HappyQuQu/foliopath/internal/systemlog"
)

type SystemLogService interface {
	List(context.Context, systemlog.Query) ([]systemlog.Event, error)
}

type systemEventResponse struct {
	ID           string  `json:"id"`
	Level        string  `json:"level"`
	Module       string  `json:"module"`
	EventCode    string  `json:"eventCode"`
	OccurredAt   string  `json:"occurredAt"`
	RequestID    *string `json:"requestId"`
	Method       *string `json:"method"`
	RoutePattern *string `json:"routePattern"`
	StatusCode   *int    `json:"statusCode"`
	DurationMS   *int64  `json:"durationMs"`
}

type systemEventPageResponse struct {
	Items      []systemEventResponse `json:"items"`
	NextCursor *string               `json:"nextCursor"`
}

func registerSystemLogRoute(mux *http.ServeMux, service SystemLogService) {
	mux.HandleFunc("GET /api/v1/system-logs", func(writer http.ResponseWriter, request *http.Request) {
		query, err := parseSystemLogQuery(request.URL.RawQuery)
		if err != nil {
			writePublicError(writer, request, http.StatusUnprocessableEntity,
				"invalid_request", "The system log query is invalid.")
			return
		}
		events, err := service.List(request.Context(), query)
		if err != nil {
			if errors.Is(err, systemlog.ErrInvalidRequest) {
				writePublicError(writer, request, http.StatusUnprocessableEntity,
					"invalid_request", "The system log query is invalid.")
				return
			}
			writeInternalError(writer, request)
			return
		}
		items := make([]systemEventResponse, 0, len(events))
		for _, event := range events {
			items = append(items, systemEventWire(event))
		}
		var next *string
		if len(events) == query.Limit {
			value := systemEventID(events[len(events)-1].ID)
			next = &value
		}
		writeJSON(writer, http.StatusOK, systemEventPageResponse{
			Items: items, NextCursor: next,
		})
	})
}

func parseSystemLogQuery(raw string) (systemlog.Query, error) {
	values, err := url.ParseQuery(raw)
	if err != nil {
		return systemlog.Query{}, systemlog.ErrInvalidRequest
	}
	for key, entries := range values {
		if (key != "level" && key != "module" && key != "cursor" &&
			key != "limit") || len(entries) != 1 {
			return systemlog.Query{}, systemlog.ErrInvalidRequest
		}
	}
	query := systemlog.Query{
		Level:  systemlog.Level(values.Get("level")),
		Module: values.Get("module"),
		Limit:  50,
	}
	if query.Level != "" && query.Level != systemlog.LevelInfo &&
		query.Level != systemlog.LevelWarning && query.Level != systemlog.LevelError {
		return systemlog.Query{}, systemlog.ErrInvalidRequest
	}
	if value := values.Get("cursor"); value != "" {
		query.BeforeID, err = parseResourceID(value, "sevt_")
		if err != nil {
			return systemlog.Query{}, systemlog.ErrInvalidRequest
		}
	}
	if value := values.Get("limit"); value != "" {
		query.Limit, err = strconv.Atoi(value)
		if err != nil {
			return systemlog.Query{}, systemlog.ErrInvalidRequest
		}
	}
	return query, nil
}

func systemEventWire(event systemlog.Event) systemEventResponse {
	response := systemEventResponse{
		ID: systemEventID(event.ID), Level: string(event.Level),
		Module: event.Module, EventCode: event.EventCode,
		OccurredAt: time.UnixMilli(event.OccurredAtMS).UTC().Format(time.RFC3339Nano),
	}
	if event.RequestID != "" {
		response.RequestID = &event.RequestID
	}
	if event.Method != "" {
		response.Method = &event.Method
	}
	if event.RoutePattern != "" {
		response.RoutePattern = &event.RoutePattern
	}
	if event.StatusCode != 0 {
		response.StatusCode = &event.StatusCode
	}
	if event.DurationMS != 0 {
		response.DurationMS = &event.DurationMS
	}
	return response
}

func systemEventID(id int64) string { return "sevt_" + strconv.FormatInt(id, 10) }
