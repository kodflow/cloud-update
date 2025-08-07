// Package entity defines the core business entities for the Cloud Update service.
package entity

import "time"

// ActionType represents the type of action to be performed.
type ActionType string

// Action types supported by the Cloud Update service.
const (
	ActionReinit        ActionType = "reinit"         // Réinitialiser cloud-init
	ActionReboot        ActionType = "reboot"         // Redémarrer le serveur
	ActionUpdate        ActionType = "update"         // Mettre à jour le système
	ActionShutdown      ActionType = "shutdown"       // Éteindre le serveur
	ActionExecuteScript ActionType = "execute_script" // Exécuter un script personnalisé
	ActionUpgrade       ActionType = "upgrade"        // Mise à niveau complète du système
	ActionRestart       ActionType = "restart"        // Redémarrer des services spécifiques
)

// WebhookRequest represents an incoming webhook request from GitHub.
type WebhookRequest struct {
	Action    ActionType        `json:"action"`
	Module    string            `json:"module,omitempty"`
	Config    map[string]string `json:"config,omitempty"`
	Timestamp int64             `json:"timestamp"`
}

// WebhookResponse represents the response to a webhook request.
type WebhookResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	JobID   string `json:"job_id,omitempty"`
}

// Job represents an asynchronous task being processed.
type Job struct {
	ID        string
	Action    ActionType
	Status    JobStatus
	StartTime time.Time
	EndTime   *time.Time
	Error     error
}

// JobStatus represents the current status of a job.
type JobStatus string

// Job status values.
const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)
