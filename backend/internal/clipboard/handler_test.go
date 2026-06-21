package clipboard_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"homelab/backend/internal/clipboard"
	"homelab/backend/internal/database"
)

func TestClipboardHTTPFlow(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)

	createResponse := doRequest(t, server, http.MethodPost, "/api/v1/clipboard-items", `{"text":"hello clipboard"}`)
	if createResponse.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, createResponse.Code, createResponse.Body.String())
	}

	var created clipboard.Item
	decodeBody(t, createResponse, &created)
	if created.ID == "" || created.Text != "hello clipboard" {
		t.Fatalf("unexpected created item: %+v", created)
	}

	listResponse := doRequest(t, server, http.MethodGet, "/api/v1/clipboard-items", "")
	if listResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, listResponse.Code, listResponse.Body.String())
	}

	var list clipboard.ListItemsResponse
	decodeBody(t, listResponse, &list)
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].ID != created.ID {
		t.Fatalf("unexpected list response: %+v", list)
	}
	if list.Page != 1 || list.PageSize != clipboard.DefaultPageSize || list.Pages != 1 || list.HasNext || list.HasPrevious {
		t.Fatalf("unexpected pagination metadata: %+v", list)
	}

	deleteResponse := doRequest(t, server, http.MethodDelete, "/api/v1/clipboard-items/"+created.ID, "")
	if deleteResponse.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, deleteResponse.Code, deleteResponse.Body.String())
	}
}

func TestClipboardListPaginatesByPage(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	for index := range 18 {
		response := doRequest(t, server, http.MethodPost, "/api/v1/clipboard-items", `{"text":"item `+strconv.Itoa(index)+`"}`)
		if response.Code != http.StatusCreated {
			t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, response.Code, response.Body.String())
		}
	}

	firstPageResponse := doRequest(t, server, http.MethodGet, "/api/v1/clipboard-items", "")
	if firstPageResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, firstPageResponse.Code, firstPageResponse.Body.String())
	}

	var firstPage clipboard.ListItemsResponse
	decodeBody(t, firstPageResponse, &firstPage)
	if len(firstPage.Items) != clipboard.DefaultPageSize || firstPage.Total != 18 || firstPage.Pages != 2 || !firstPage.HasNext || firstPage.HasPrevious {
		t.Fatalf("unexpected first page: %+v", firstPage)
	}

	secondPageResponse := doRequest(t, server, http.MethodGet, "/api/v1/clipboard-items?page=2&pageSize=15", "")
	if secondPageResponse.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, secondPageResponse.Code, secondPageResponse.Body.String())
	}

	var secondPage clipboard.ListItemsResponse
	decodeBody(t, secondPageResponse, &secondPage)
	if len(secondPage.Items) != 3 || secondPage.Page != 2 || secondPage.Total != 18 || secondPage.HasNext || !secondPage.HasPrevious {
		t.Fatalf("unexpected second page: %+v", secondPage)
	}
}

func TestCreateClipboardItemValidatesText(t *testing.T) {
	t.Parallel()

	server := newTestServer(t)
	response := doRequest(t, server, http.MethodPost, "/api/v1/clipboard-items", `{"text":"   "}`)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status %d, got %d: %s", http.StatusUnprocessableEntity, response.Code, response.Body.String())
	}
}

func newTestServer(t *testing.T) http.Handler {
	t.Helper()

	db, err := database.Open(context.Background(), t.TempDir()+"/homelab-test.db")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() {
		db.Close()
	})

	repository := clipboard.NewRepository(db)
	service := clipboard.NewService(repository)
	handler := clipboard.NewHandler(service)

	mux := http.NewServeMux()
	handler.Register(mux)

	return mux
}

func doRequest(t *testing.T, handler http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		request.Header.Set("Content-Type", "application/json")
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func decodeBody(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()

	if err := json.NewDecoder(response.Body).Decode(target); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}
