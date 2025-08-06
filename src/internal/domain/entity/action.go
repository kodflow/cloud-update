package entity

import "time"

type ActionType string

const (
	ActionReinit        ActionType = "reinit"         // Réinitialiser cloud-init
	ActionReboot        ActionType = "reboot"         // Redémarrer le serveur
	ActionUpdate        ActionType = "update"         // Mettre à jour le système
	ActionShutdown      ActionType = "shutdown"       // Éteindre le serveur
	ActionExecuteScript ActionType = "execute_script" // Exécuter un script personnalisé
	ActionUpgrade       ActionType = "upgrade"        // Mise à niveau complète du système
	ActionRestart       ActionType = "restart"        // Redémarrer des services spécifiques
)

type WebhookRequest struct {
	Action    ActionType        `json:"action"`
	Module    string            `json:"module,omitempty"`
	Config    map[string]string `json:"config,omitempty"`
	Timestamp int64             `json:"timestamp"`
}

type WebhookResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	JobID   string `json:"job_id,omitempty"`
}

type Job struct {
	ID        string
	Action    ActionType
	Status    JobStatus
	StartTime time.Time
	EndTime   *time.Time
	Error     error
}

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
)
