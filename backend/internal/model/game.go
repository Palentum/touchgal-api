package model

import "time"

type GameSearchItem struct {
	Name     string `json:"name"`
	UniqueID string `json:"uniqueId"`
}

type GameSearchResult struct {
	Items      []GameSearchItem `json:"items"`
	Pagination Pagination       `json:"pagination"`
}

type Pagination struct {
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
	Total   int  `json:"total"`
	HasMore bool `json:"hasMore"`
}

type GameDetail struct {
	UniqueID           string        `json:"uniqueId"`
	Name               string        `json:"name"`
	Aliases            []string      `json:"aliases"`
	Introduction       string        `json:"introduction"`
	BannerURL          string        `json:"bannerUrl"`
	Type               []string      `json:"type"`
	Platform           []string      `json:"platform"`
	Language           []string      `json:"language"`
	Tags               []string      `json:"tags"`
	PublishTime        time.Time     `json:"publishTime"`
	ReleaseDate        string        `json:"releaseDate"`
	UpdatedAt          time.Time     `json:"updatedAt"`
	ResourceUpdateTime time.Time     `json:"resourceUpdateTime"`
	Companies          []CompanyView `json:"companies"`
	Rating             RatingView    `json:"rating"`
	TouchGalURL        string        `json:"touchgalUrl"`
}

type CompanyView struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases"`
}

type RatingView struct {
	Average   float64       `json:"average"`
	Count     int           `json:"count"`
	Recommend RecommendView `json:"recommend"`
}

type RecommendView struct {
	StrongNo  int `json:"strongNo"`
	No        int `json:"no"`
	Neutral   int `json:"neutral"`
	Yes       int `json:"yes"`
	StrongYes int `json:"strongYes"`
}

type CleanGame struct {
	UniqueID          string
	SourcePatchID     int
	Name              string
	Introduction      string
	BannerURL         string
	Released          string
	ContentLimit      string
	Types             []string
	Languages         []string
	Platforms         []string
	SourceCreatedAt   time.Time
	SourceUpdatedAt   time.Time
	ResourceUpdatedAt time.Time
}

type TagData struct {
	Name    string
	Aliases []string
	Source  string
}

type CompanyData struct {
	Name             string
	Aliases          []string
	OfficialWebsites []string
	PrimaryLanguages []string
	ParentBrands     []string
}

type RatingData struct {
	AverageOverall float64
	Count          int
	RecStrongNo    int
	RecNo          int
	RecNeutral     int
	RecYes         int
	RecStrongYes   int
	Histogram      map[string]int
}
