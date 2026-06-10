package model

import (
	"strconv"
	"time"
)

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

type RatingHistogram struct {
	Score1  int `json:"1"`
	Score2  int `json:"2"`
	Score3  int `json:"3"`
	Score4  int `json:"4"`
	Score5  int `json:"5"`
	Score6  int `json:"6"`
	Score7  int `json:"7"`
	Score8  int `json:"8"`
	Score9  int `json:"9"`
	Score10 int `json:"10"`
}

func (h RatingHistogram) MarshalJSON() ([]byte, error) {
	buf := make([]byte, 0, 64)
	buf = append(buf, '{')
	first := true
	appendBucket := func(score string, count int) {
		if count == 0 {
			return
		}
		if !first {
			buf = append(buf, ',')
		}
		first = false
		buf = append(buf, '"')
		buf = append(buf, score...)
		buf = append(buf, '"', ':')
		buf = strconv.AppendInt(buf, int64(count), 10)
	}
	appendBucket("1", h.Score1)
	appendBucket("2", h.Score2)
	appendBucket("3", h.Score3)
	appendBucket("4", h.Score4)
	appendBucket("5", h.Score5)
	appendBucket("6", h.Score6)
	appendBucket("7", h.Score7)
	appendBucket("8", h.Score8)
	appendBucket("9", h.Score9)
	appendBucket("10", h.Score10)
	buf = append(buf, '}')
	return buf, nil
}

type RatingData struct {
	AverageOverall float64
	Count          int
	RecStrongNo    int
	RecNo          int
	RecNeutral     int
	RecYes         int
	RecStrongYes   int
	Histogram      RatingHistogram
}
