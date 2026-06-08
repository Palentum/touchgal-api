package model

import (
	"time"

	"github.com/google/uuid"
)

type RequestLog struct {
	TokenID       *uuid.UUID
	UserID        *uuid.UUID
	ApplicationID *uuid.UUID
	Method        string
	Path          string
	Route         string
	StatusCode    int
	LatencyMS     int
	IP            string
	UserAgent     string
	Origin        string
	Referer       string
}

type StatsSummary struct {
	TotalRequests   int `json:"totalRequests"`
	SuccessRequests int `json:"successRequests"`
	ErrorRequests   int `json:"errorRequests"`
	AvgLatencyMS    int `json:"avgLatencyMs"`
	UniqueOrigins   int `json:"uniqueOrigins"`
	UniqueIPs       int `json:"uniqueIPs"`
}

type StatsTrend struct {
	Date            string `json:"date"`
	TotalRequests   int    `json:"totalRequests"`
	SuccessRequests int    `json:"successRequests"`
	ErrorRequests   int    `json:"errorRequests"`
}

type StatsSource struct {
	Origin      string `json:"origin"`
	RefererHost string `json:"refererHost"`
	Requests    int    `json:"requests"`
}

type StatsEndpoint struct {
	Route        string  `json:"route"`
	Requests     int     `json:"requests"`
	AvgLatencyMS int     `json:"avgLatencyMs"`
	ErrorRate    float64 `json:"errorRate"`
}

type SyncRun struct {
	ID                 uuid.UUID  `json:"id"`
	Mode               string     `json:"mode"`
	Status             string     `json:"status"`
	StartedAt          time.Time  `json:"startedAt"`
	FinishedAt         *time.Time `json:"finishedAt,omitempty"`
	SourceMaxUpdatedAt *time.Time `json:"sourceMaxUpdatedAt,omitempty"`
	GamesSeen          int        `json:"gamesSeen"`
	GamesUpserted      int        `json:"gamesUpserted"`
	GamesDeleted       int        `json:"gamesDeleted"`
	ErrorMessage       string     `json:"errorMessage"`
}
