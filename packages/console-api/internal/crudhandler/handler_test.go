package crudhandler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	managementv1 "code-code.internal/go-contract/platform/management/v1"
	skillv1 "code-code.internal/go-contract/platform/skill/v1"
	"google.golang.org/protobuf/proto"
)

func TestRegisterListEndpoint(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, stubService{}, Config[*managementv1.SkillListItem, *skillv1.Skill, *managementv1.UpsertSkillRequest]{
		ResourceName: "skills",
		WrapList:     func(items []*managementv1.SkillListItem) proto.Message { return &managementv1.ListSkillsResponse{Items: items} },
		NewRequest:   func() *managementv1.UpsertSkillRequest { return &managementv1.UpsertSkillRequest{} },
	})

	request := httptest.NewRequest(http.MethodGet, "/api/skills", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var payload struct {
		Items []struct {
			SkillID string `json:"skillId"`
		} `json:"items"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(payload.Items) != 1 || payload.Items[0].SkillID != "concise-reply" {
		t.Fatalf("payload = %#v", payload.Items)
	}
}

func TestRegisterCreateAndDeleteEndpoints(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, stubService{}, Config[*managementv1.SkillListItem, *skillv1.Skill, *managementv1.UpsertSkillRequest]{
		ResourceName: "skills",
		WrapList:     func(items []*managementv1.SkillListItem) proto.Message { return &managementv1.ListSkillsResponse{Items: items} },
		NewRequest:   func() *managementv1.UpsertSkillRequest { return &managementv1.UpsertSkillRequest{} },
	})

	create := httptest.NewRequest(http.MethodPost, "/api/skills", strings.NewReader(`{"skillId":"translator","name":"Translator","description":"Translate text","content":"Translate faithfully."}`))
	create.Header.Set("Content-Type", "application/json")
	createRecorder := httptest.NewRecorder()
	mux.ServeHTTP(createRecorder, create)
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want 201, body=%s", createRecorder.Code, createRecorder.Body.String())
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/skills/translator", nil)
	deleteRecorder := httptest.NewRecorder()
	mux.ServeHTTP(deleteRecorder, deleteReq)
	if deleteRecorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want 200, body=%s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestRegisterMethodNotAllowed(t *testing.T) {
	mux := http.NewServeMux()
	Register(mux, stubService{}, Config[*managementv1.SkillListItem, *skillv1.Skill, *managementv1.UpsertSkillRequest]{
		ResourceName: "skills",
		WrapList:     func(items []*managementv1.SkillListItem) proto.Message { return &managementv1.ListSkillsResponse{Items: items} },
		NewRequest:   func() *managementv1.UpsertSkillRequest { return &managementv1.UpsertSkillRequest{} },
	})

	request := httptest.NewRequest(http.MethodPatch, "/api/skills", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", recorder.Code)
	}
}

type stubService struct{}

func (stubService) List(context.Context) ([]*managementv1.SkillListItem, error) {
	return []*managementv1.SkillListItem{{SkillId: "concise-reply", Name: "Concise Reply"}}, nil
}

func (stubService) Get(context.Context, string) (*skillv1.Skill, error) {
	return &skillv1.Skill{SkillId: "concise-reply", Name: "Concise Reply"}, nil
}

func (stubService) Create(_ context.Context, request *managementv1.UpsertSkillRequest) (*skillv1.Skill, error) {
	return &skillv1.Skill{SkillId: request.GetSkillId(), Name: request.GetName()}, nil
}

func (stubService) Update(_ context.Context, skillID string, request *managementv1.UpsertSkillRequest) (*skillv1.Skill, error) {
	return &skillv1.Skill{SkillId: skillID, Name: request.GetName()}, nil
}

func (stubService) Delete(context.Context, string) error { return nil }
