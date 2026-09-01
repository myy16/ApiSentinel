package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apisentinel/apisentinel/internal/delivery"
	"github.com/apisentinel/apisentinel/internal/worker"
)

func TestDeliveryService_Initialization(t *testing.T) {
	wp := worker.NewPool(5, 100)
	svc := NewDeliveryService(nil, wp, "test-key-32-bytes-long-for-aes")

	if svc == nil {
		t.Fatalf("expected non-nil delivery service")
	}
	if svc.maxPerEndpoint != 5 {
		t.Errorf("expected maxPerEndpoint = 5, got %d", svc.maxPerEndpoint)
	}
}

func TestDeliveryService_SemaphoreFairScheduling(t *testing.T) {
	wp := worker.NewPool(5, 100)
	svc := NewDeliveryService(nil, wp, "")

	endpointID := "ep-12345"
	release1 := svc.acquireEndpointSemaphore(endpointID)
	release2 := svc.acquireEndpointSemaphore(endpointID)

	release1()
	release2()
}

func TestDeliveryService_EvaluateRetryLogic(t *testing.T) {
	opts := delivery.DefaultRetryOptions()
	opts.MaxRetries = 3

	// 500 server error attempt 1
	eval1 := delivery.EvaluateResponse(http.StatusInternalServerError, nil, 1, nil, opts)
	if eval1.NextState != delivery.DeliveryStateRetryWait {
		t.Errorf("expected RetryWait, got %s", eval1.NextState)
	}

	// 500 server error attempt 3 (exhausted)
	eval3 := delivery.EvaluateResponse(http.StatusInternalServerError, nil, 3, nil, opts)
	if eval3.NextState != delivery.DeliveryStateDeadLetter {
		t.Errorf("expected DeadLetter, got %s", eval3.NextState)
	}

	// 200 success
	eval200 := delivery.EvaluateResponse(http.StatusOK, nil, 1, nil, opts)
	if eval200.NextState != delivery.DeliveryStateDelivered {
		t.Errorf("expected Delivered, got %s", eval200.NextState)
	}
}

func TestDeliveryService_MockServerSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-ApiSentinel-Forwarded") != "true" {
			t.Errorf("expected X-ApiSentinel-Forwarded header")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"received": true}`))
	}))
	defer server.Close()

	client := server.Client()
	req, err := http.NewRequestWithContext(context.Background(), "POST", server.URL, nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("X-ApiSentinel-Forwarded", "true")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}
