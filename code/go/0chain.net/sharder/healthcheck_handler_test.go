package sharder_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"0chain.net/sharder"
)

func TestHealthCheckWriter(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/_health_check", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(sharder.HealthCheckWriter)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}
}
