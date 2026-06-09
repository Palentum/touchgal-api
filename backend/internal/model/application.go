package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	ApplicationPending  = "pending"
	ApplicationApproved = "approved"
	ApplicationRejected = "rejected"
	ApplicationRevoked  = "revoked"
)

type Application struct {
	ID                    uuid.UUID  `json:"id"`
	UserID                uuid.UUID  `json:"userId"`
	ApplicantName         string     `json:"applicantName"`
	ProjectName           string     `json:"projectName"`
	ProjectURL            string     `json:"projectUrl"`
	ExpectedDailyRequests int        `json:"expectedDailyRequests"`
	UsageScenario         string     `json:"usageScenario"`
	Status                string     `json:"status"`
	DefaultMinuteLimit    int        `json:"defaultMinuteLimit"`
	DefaultDailyLimit     int        `json:"defaultDailyLimit"`
	ReviewNote            string     `json:"reviewNote"`
	ReviewedBy            *uuid.UUID `json:"reviewedBy,omitempty"`
	ReviewedAt            *time.Time `json:"reviewedAt,omitempty"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type CreateApplicationInput struct {
	ApplicantName         string
	ProjectName           string
	ProjectURL            string
	ExpectedDailyRequests int
	UsageScenario         string
}
