package httpjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const defaultMaxJSONBodyBytes = 1 << 20

const corsAllowedHeaders = "Accept,Content-Type," +
	"Connect-Protocol-Version,Connect-Timeout-Ms,Connect-Accept-Encoding,Connect-Content-Encoding," +
	"Grpc-Timeout,Grpc-Encoding,Grpc-Accept-Encoding,X-Grpc-Web,X-User-Agent,Authorization"

// ErrorResponse describes one structured HTTP error payload.
type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes one JSON response with the provided status code.
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if payload == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(payload)
}

var protoJSONMarshaler = protojson.MarshalOptions{EmitUnpopulated: true}

// WriteProtoJSON writes one proto message as JSON with camelCase field names.
func WriteProtoJSON(w http.ResponseWriter, status int, msg proto.Message) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if msg == nil {
		return
	}
	data, err := protoJSONMarshaler.Marshal(msg)
	if err != nil {
		return
	}
	_, _ = w.Write(data)
	_, _ = w.Write([]byte("\n"))
}

var protoJSONUnmarshaler = protojson.UnmarshalOptions{DiscardUnknown: true}

// DecodeProtoJSON decodes one JSON request body into a proto message.
func DecodeProtoJSON(r *http.Request, msg proto.Message) error {
	if r == nil {
		return fmt.Errorf("httpjson: request is nil")
	}
	if msg == nil {
		return fmt.Errorf("httpjson: target is nil")
	}
	limited := &io.LimitedReader{R: r.Body, N: defaultMaxJSONBodyBytes + 1}
	body, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("httpjson: read json body: %w", err)
	}
	if int64(len(body)) > defaultMaxJSONBodyBytes {
		return fmt.Errorf("httpjson: json body exceeds %d bytes", defaultMaxJSONBodyBytes)
	}
	if err := protoJSONUnmarshaler.Unmarshal(body, msg); err != nil {
		return fmt.Errorf("httpjson: decode json: %w", err)
	}
	return nil
}

// WriteError writes one structured JSON error response.
func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, ErrorResponse{
		Code:    code,
		Message: message,
	})
}

// WriteServiceError writes one service error with normalized HTTP status and
// safe message body.
func WriteServiceError(w http.ResponseWriter, fallbackStatus int, code string, err error) {
	status := StatusForServiceError(err, fallbackStatus)
	WriteError(w, status, code, MessageForServiceError(err, status))
}

// StatusForServiceError maps one service error to one HTTP status code.
func StatusForServiceError(err error, fallbackStatus int) int {
	if err == nil {
		if fallbackStatus >= http.StatusBadRequest {
			return fallbackStatus
		}
		return http.StatusInternalServerError
	}
	if grpcStatus, ok := grpcstatus.FromError(err); ok {
		switch grpcStatus.Code() {
		case codes.OK:
			return fallbackStatus
		case codes.InvalidArgument, codes.OutOfRange:
			return http.StatusBadRequest
		case codes.NotFound:
			return http.StatusNotFound
		case codes.AlreadyExists, codes.Aborted:
			return http.StatusConflict
		case codes.FailedPrecondition:
			return http.StatusConflict
		case codes.PermissionDenied:
			return http.StatusForbidden
		case codes.Unauthenticated:
			return http.StatusUnauthorized
		case codes.ResourceExhausted:
			return http.StatusTooManyRequests
		case codes.Unavailable, codes.DeadlineExceeded:
			return http.StatusServiceUnavailable
		default:
			return http.StatusInternalServerError
		}
	}

	if fallbackStatus >= http.StatusInternalServerError {
		return fallbackStatus
	}
	return http.StatusInternalServerError
}

// MessageForServiceError normalizes the client-facing message.
func MessageForServiceError(err error, status int) string {
	if err == nil {
		if text := strings.TrimSpace(http.StatusText(status)); text != "" {
			return strings.ToLower(text)
		}
		return "request failed"
	}
	if status >= http.StatusInternalServerError {
		return "internal server error"
	}
	if grpcStatus, ok := grpcstatus.FromError(err); ok {
		if message := strings.TrimSpace(grpcStatus.Message()); message != "" {
			return message
		}
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		if text := strings.TrimSpace(http.StatusText(status)); text != "" {
			return strings.ToLower(text)
		}
		return "request failed"
	}
	return message
}

// DecodeJSON decodes one JSON request body and rejects unknown fields.
func DecodeJSON(r *http.Request, out any) error {
	if r == nil {
		return fmt.Errorf("httpjson: request is nil")
	}
	if out == nil {
		return fmt.Errorf("httpjson: target is nil")
	}
	limited := &io.LimitedReader{R: r.Body, N: defaultMaxJSONBodyBytes + 1}
	body, err := io.ReadAll(limited)
	if err != nil {
		return fmt.Errorf("httpjson: read json body: %w", err)
	}
	if int64(len(body)) > defaultMaxJSONBodyBytes {
		return fmt.Errorf("httpjson: json body exceeds %d bytes", defaultMaxJSONBodyBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return fmt.Errorf("httpjson: decode json: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("httpjson: decode json: multiple json values")
	}
	return nil
}

// WithCORS applies one permissive development CORS policy suitable for the
// internal console during local development.
func WithCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", corsAllowedHeaders)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
