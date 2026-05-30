package workspace

import "time"

type Status string

const (
	StatusCreating Status = "CREATING"
	StatusStarting Status = "STARTING"
	StatusRunning  Status = "RUNNING"
	StatusStopping Status = "STOPPING"
	StatusStopped  Status = "STOPPED"
	StatusDeleted  Status = "DELETED"
)

type Resources struct {
	CPU       float64 `firestore:"cpu" json:"cpu"`
	MemoryGB  float64 `firestore:"memoryGb" json:"memoryGb"`
	StorageGB float64 `firestore:"storageGb" json:"storageGb"`
}

type Workspace struct {
	ID         string    `firestore:"id" json:"id"`
	TenantID   string    `firestore:"tenantId" json:"tenantId"`
	UserID     string    `firestore:"userId" json:"userId"`
	Image      string    `firestore:"image" json:"image"`
	Resources  Resources `firestore:"resources" json:"resources"`
	Status     Status    `firestore:"status" json:"status"`
	CreatedAt  time.Time `firestore:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time `firestore:"updatedAt" json:"updatedAt"`
	LastActive time.Time `firestore:"lastActiveAt,omitempty" json:"lastActiveAt,omitempty"`
}
