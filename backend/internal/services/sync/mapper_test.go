package syncsvc

import (
	"reflect"
	"testing"
	"time"

	"github.com/touchgal/developer/backend/internal/model"
)

func TestMapSourceResource(t *testing.T) {
	created := time.Date(2026, 6, 14, 10, 0, 0, 0, time.UTC)
	updated := created.Add(2 * time.Hour)

	tests := []struct {
		name         string
		src          SourceResource
		gameUniqueID string
		want         model.CleanResourceEntry
		wantOK       bool
	}{
		{
			name: "galgame resource maps to clean resource metadata",
			src: SourceResource{
				ResourceID:   42,
				Name:         " 汉化补丁 ",
				Introduction: "补丁简介",
				Categories:   []string{"patch", "patch", ""},
				Section:      "galgame",
				Sizes:        []string{"1.2 GB", "1.2 GB", ""},
				CreatedAt:    created,
				UpdatedAt:    updated,
			},
			gameUniqueID: "abcd1234",
			want: model.CleanResourceEntry{
				SourceResourceID: 42,
				GameUniqueID:     "abcd1234",
				Name:             "汉化补丁",
				Introduction:     "补丁简介",
				Categories:       []string{"patch"},
				ResourceType:     model.ResourceTypeResource,
				Sizes:            []string{"1.2 GB"},
				PublishedAt:      created,
				SourceUpdatedAt:  updated,
			},
			wantOK: true,
		},
		{
			name: "patch section maps to clean patch resource type",
			src: SourceResource{
				ResourceID: 7,
				Section:    " patch ",
				CreatedAt:  created,
				UpdatedAt:  updated,
			},
			gameUniqueID: " efgh5678 ",
			want: model.CleanResourceEntry{
				SourceResourceID: 7,
				GameUniqueID:     "efgh5678",
				Categories:       []string{},
				ResourceType:     model.ResourceTypePatch,
				Sizes:            []string{},
				PublishedAt:      created,
				SourceUpdatedAt:  updated,
			},
			wantOK: true,
		},
		{
			name: "empty name introduction categories and sizes are accepted",
			src: SourceResource{
				ResourceID: 8,
				Section:    "galgame",
				CreatedAt:  created,
				UpdatedAt:  updated,
			},
			gameUniqueID: "ijkl9012",
			want: model.CleanResourceEntry{
				SourceResourceID: 8,
				GameUniqueID:     "ijkl9012",
				Categories:       []string{},
				ResourceType:     model.ResourceTypeResource,
				Sizes:            []string{},
				PublishedAt:      created,
				SourceUpdatedAt:  updated,
			},
			wantOK: true,
		},
		{
			name:         "unknown section is rejected",
			src:          SourceResource{ResourceID: 9, Section: "unknown"},
			gameUniqueID: "abcd1234",
			wantOK:       false,
		},
		{
			name:         "non-positive resource ID is rejected",
			src:          SourceResource{ResourceID: 0, Section: "galgame"},
			gameUniqueID: "abcd1234",
			wantOK:       false,
		},
		{
			name:         "empty game unique ID is rejected",
			src:          SourceResource{ResourceID: 10, Section: "galgame"},
			gameUniqueID: "   ",
			wantOK:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := MapSourceResource(tt.src, tt.gameUniqueID)
			if ok != tt.wantOK {
				t.Fatalf("expected ok=%v, got %v", tt.wantOK, ok)
			}
			if !tt.wantOK {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("unexpected mapped resource:\n got: %#v\nwant: %#v", got, tt.want)
			}
		})
	}
}
