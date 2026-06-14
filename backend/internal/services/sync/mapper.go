package syncsvc

import (
	"strings"
	"time"

	"github.com/touchgal/developer/backend/internal/model"
)

type SourceGame struct {
	ID                int
	UniqueID          string
	Name              string
	Introduction      string
	Banner            string
	Released          string
	ContentLimit      string
	Types             []string
	Languages         []string
	Platforms         []string
	CreatedAt         time.Time
	UpdatedAt         time.Time
	ResourceUpdatedAt time.Time
}

type SourceResource struct {
	PatchID      int
	ResourceID   int
	Name         string
	Introduction string
	Categories   []string
	Section      string
	Sizes        []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func MapSourceGame(src SourceGame, defaultContentPolicy string) model.CleanGame {
	contentLimit := strings.TrimSpace(src.ContentLimit)
	if contentLimit == "" && defaultContentPolicy != "" && defaultContentPolicy != "all" {
		contentLimit = defaultContentPolicy
	}
	return model.CleanGame{
		UniqueID:          strings.TrimSpace(src.UniqueID),
		SourcePatchID:     src.ID,
		Name:              strings.TrimSpace(src.Name),
		Introduction:      src.Introduction,
		BannerURL:         src.Banner,
		Released:          defaultString(strings.TrimSpace(src.Released), "unknown"),
		ContentLimit:      contentLimit,
		Types:             cleanStrings(src.Types),
		Languages:         cleanStrings(src.Languages),
		Platforms:         cleanStrings(src.Platforms),
		SourceCreatedAt:   src.CreatedAt,
		SourceUpdatedAt:   src.UpdatedAt,
		ResourceUpdatedAt: src.ResourceUpdatedAt,
	}
}

func MapSourceResource(src SourceResource, gameUniqueID string) (model.CleanResourceEntry, bool) {
	gameUniqueID = strings.TrimSpace(gameUniqueID)
	section := strings.TrimSpace(src.Section)
	if src.ResourceID <= 0 || gameUniqueID == "" {
		return model.CleanResourceEntry{}, false
	}

	var resourceType string
	switch section {
	case "galgame":
		resourceType = model.ResourceTypeResource
	case "patch":
		resourceType = model.ResourceTypePatch
	default:
		return model.CleanResourceEntry{}, false
	}

	return model.CleanResourceEntry{
		SourceResourceID: src.ResourceID,
		GameUniqueID:     gameUniqueID,
		Name:             strings.TrimSpace(src.Name),
		Introduction:     src.Introduction,
		Categories:       cleanStrings(src.Categories),
		ResourceType:     resourceType,
		Sizes:            cleanStrings(src.Sizes),
		PublishedAt:      src.CreatedAt,
		SourceUpdatedAt:  src.UpdatedAt,
	}, true
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
