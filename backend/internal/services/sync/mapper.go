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
