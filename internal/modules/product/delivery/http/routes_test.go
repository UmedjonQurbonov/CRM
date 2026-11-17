package http_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestChiProductRoutes(t *testing.T) {
	r := chi.NewRouter()

	var hitUploadImage bool
	var hitRestock bool
	var hitUpdateProduct bool
	var hitProductImage bool
	var capturedID string

	r.Route("/api/v1", func(apiRouter chi.Router) {
		apiRouter.Post("/products/upload-image", func(w http.ResponseWriter, r *http.Request) {
			hitUploadImage = true
			w.WriteHeader(http.StatusOK)
		})
		apiRouter.Put("/products/{id}", func(w http.ResponseWriter, r *http.Request) {
			hitUpdateProduct = true
			capturedID = chi.URLParam(r, "id")
			w.WriteHeader(http.StatusOK)
		})
		apiRouter.Post("/products/{id}/restock", func(w http.ResponseWriter, r *http.Request) {
			hitRestock = true
			capturedID = chi.URLParam(r, "id")
			w.WriteHeader(http.StatusOK)
		})
		apiRouter.Post("/products/{id}/image", func(w http.ResponseWriter, r *http.Request) {
			hitProductImage = true
			capturedID = chi.URLParam(r, "id")
			w.WriteHeader(http.StatusOK)
		})
	})

	// Test 1: POST /api/v1/products/upload-image
	req1 := httptest.NewRequest(http.MethodPost, "/api/v1/products/upload-image", nil)
	rec1 := httptest.NewRecorder()
	r.ServeHTTP(rec1, req1)
	if rec1.Code != http.StatusOK || !hitUploadImage {
		t.Fatalf("expected 200 and hitUploadImage=true, got %d", rec1.Code)
	}

	// Test 2: POST /api/v1/products/123-abc/restock
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/products/123-abc/restock", nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK || !hitRestock || capturedID != "123-abc" {
		t.Fatalf("expected 200, hitRestock=true, capturedID=123-abc, got %d and %s", rec2.Code, capturedID)
	}

	// Test 3: POST /api/v1/products/456-def/image
	req3 := httptest.NewRequest(http.MethodPost, "/api/v1/products/456-def/image", nil)
	rec3 := httptest.NewRecorder()
	r.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusOK || !hitProductImage || capturedID != "456-def" {
		t.Fatalf("expected 200, hitProductImage=true, capturedID=456-def, got %d and %s", rec3.Code, capturedID)
	}

	// Test 4: PUT /api/v1/products/789-ghi
	req4 := httptest.NewRequest(http.MethodPut, "/api/v1/products/789-ghi", nil)
	rec4 := httptest.NewRecorder()
	r.ServeHTTP(rec4, req4)
	if rec4.Code != http.StatusOK || !hitUpdateProduct || capturedID != "789-ghi" {
		t.Fatalf("expected 200, hitUpdateProduct=true, capturedID=789-ghi, got %d and %s", rec4.Code, capturedID)
	}
}
