package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"cdk-airdrop-station/server/internal/model"
)

type envelope struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Code    string      `json:"code,omitempty"`
	Message string      `json:"message,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope{Success: status < 400, Data: payload}); err != nil {
		slog.Error("encode json response failed", "error", err)
	}
}

func writeError(w http.ResponseWriter, err error) {
	var appErr *model.AppError
	if errors.As(err, &appErr) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(appErr.Status)
		_ = json.NewEncoder(w).Encode(envelope{Success: false, Code: appErr.Code, Message: appErr.Message})
		return
	}
	if errors.Is(err, context.DeadlineExceeded) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusGatewayTimeout)
		_ = json.NewEncoder(w).Encode(envelope{Success: false, Code: "TIMEOUT", Message: "请求超时"})
		return
	}
	slog.Error("unknown error occurred", "error", err)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(envelope{Success: false, Code: "INTERNAL_ERROR", Message: "服务器发生了未预期错误"})
}
