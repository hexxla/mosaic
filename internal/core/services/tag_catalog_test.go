package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/sploitzberg/mosaic/internal/core/domain"
	"github.com/sploitzberg/mosaic/internal/core/services"
)

type stubTagCatalog struct {
	list     domain.DistinctTagsResult
	counts   domain.TagCountsResult
	listErr  error
	countErr error
}

func (s *stubTagCatalog) ListDistinctTags(_ context.Context) (domain.DistinctTagsResult, error) {
	if s.listErr != nil {
		return domain.DistinctTagsResult{}, s.listErr
	}
	return s.list, nil
}

func (s *stubTagCatalog) TagCounts(_ context.Context) (domain.TagCountsResult, error) {
	if s.countErr != nil {
		return domain.TagCountsResult{}, s.countErr
	}
	return s.counts, nil
}

func TestTagCatalogService_ListDistinctTags_propagates(t *testing.T) {
	t.Parallel()
	wantErr := errors.New("boom")
	stub := &stubTagCatalog{listErr: wantErr}
	svc := services.NewTagCatalogService(stub)
	_, err := svc.ListDistinctTags(t.Context())
	if !errors.Is(err, wantErr) {
		t.Fatalf("expected boom: %v", err)
	}
}

func TestTagCatalogService_ListDistinctTags_ok(t *testing.T) {
	t.Parallel()
	stub := &stubTagCatalog{
		list: domain.DistinctTagsResult{Tags: []string{"a", "b"}},
	}
	svc := services.NewTagCatalogService(stub)
	out, err := svc.ListDistinctTags(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Tags) != 2 || out.Tags[0] != "a" {
		t.Fatalf("tags: %+v", out.Tags)
	}
}

func TestTagCatalogService_nilCatalog(t *testing.T) {
	t.Parallel()
	svc := services.NewTagCatalogService(nil)
	_, err := svc.ListDistinctTags(t.Context())
	if err == nil {
		t.Fatal("expected error")
	}
}
