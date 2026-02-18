package handler

import (
	"errors"
	"testing"

	"github.com/UnAfraid/wg-ui/pkg/api/internal/model"
)

func TestResultAndErrorToDataloaderResultOrdersByKeys(t *testing.T) {
	keys := []string{"backend-2", "backend-1", "backend-missing"}

	values := []*model.Backend{
		{
			ID: model.StringID(model.IdKindBackend, "backend-1"),
		},
		{
			ID: model.StringID(model.IdKindBackend, "backend-2"),
		},
	}

	results := resultAndErrorToDataloaderResult(keys, values, func(item *model.Backend) string {
		if item == nil {
			return ""
		}
		return item.ID.Value
	}, nil)

	if len(results) != len(keys) {
		t.Fatalf("expected %d results, got %d", len(keys), len(results))
	}

	if results[0].Data == nil || results[0].Data.ID.Value != "backend-2" {
		t.Fatalf("expected first result to map to backend-2, got %#v", results[0].Data)
	}

	if results[1].Data == nil || results[1].Data.ID.Value != "backend-1" {
		t.Fatalf("expected second result to map to backend-1, got %#v", results[1].Data)
	}

	if results[2].Data != nil {
		t.Fatalf("expected missing key to return nil data, got %#v", results[2].Data)
	}
}

func TestResultAndErrorToDataloaderResultPropagatesErrors(t *testing.T) {
	keys := []string{"a", "b"}
	expectedErr := errors.New("boom")

	results := resultAndErrorToDataloaderResult(keys, []*model.Backend{}, func(item *model.Backend) string {
		if item == nil {
			return ""
		}
		return item.ID.Value
	}, expectedErr)

	if len(results) != len(keys) {
		t.Fatalf("expected %d results, got %d", len(keys), len(results))
	}

	for i, result := range results {
		if !errors.Is(result.Error, expectedErr) {
			t.Fatalf("result %d: expected error %v, got %v", i, expectedErr, result.Error)
		}
	}
}
