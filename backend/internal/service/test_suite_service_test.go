package service

import (
	"context"
	"fmt"
	"testing"
)

func TestTestSuiteService_ValidationLimits(t *testing.T) {
	svc := NewTestSuiteService(nil, nil)

	// 1. Empty name
	_, err := svc.CreateSuite(context.Background(), CreateTestSuiteParams{
		ProjectID:  "11111111-1111-1111-1111-111111111111",
		Name:       "",
		RequestIDs: []string{"22222222-2222-2222-2222-222222222222"},
	})
	if err == nil || err.Error() != "test paketi adı zorunludur" {
		t.Fatalf("Expected empty name error, got: %v", err)
	}

	// 2. Empty requests
	_, err = svc.CreateSuite(context.Background(), CreateTestSuiteParams{
		ProjectID:  "11111111-1111-1111-1111-111111111111",
		Name:       "Test Suite",
		RequestIDs: []string{},
	})
	if err == nil || err.Error() != "en az bir istek seçilmelidir" {
		t.Fatalf("Expected empty requests error, got: %v", err)
	}

	// 3. Exceeds 50 requests limit (#8)
	var tooManyReqs []string
	for i := 0; i < 51; i++ {
		tooManyReqs = append(tooManyReqs, fmt.Sprintf("req-%d", i))
	}
	_, err = svc.CreateSuite(context.Background(), CreateTestSuiteParams{
		ProjectID:  "11111111-1111-1111-1111-111111111111",
		Name:       "Test Suite Overload",
		RequestIDs: tooManyReqs,
	})
	if err == nil || err.Error() != "bir test paketine en fazla 50 istek eklenebilir" {
		t.Fatalf("Expected max 50 requests limit error, got: %v", err)
	}
}
