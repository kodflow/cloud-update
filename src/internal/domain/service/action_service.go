// Package service implements the business logic for the Cloud Update service.
package service

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/kodflow/cloud-update/src/internal/domain/entity"
	"github.com/kodflow/cloud-update/src/internal/infrastructure/system"
)

// rebootDelay is the delay before executing a reboot.
// This is a variable so it can be modified in tests.
var (
	rebootDelay   = 10 * time.Second
	rebootDelayMu sync.RWMutex
)

// getRebootDelay returns the current reboot delay value safely.
func getRebootDelay() time.Duration {
	rebootDelayMu.RLock()
	defer rebootDelayMu.RUnlock()
	return rebootDelay
}

// setRebootDelay sets the reboot delay value safely.
func setRebootDelay(d time.Duration) {
	rebootDelayMu.Lock()
	defer rebootDelayMu.Unlock()
	rebootDelay = d
}

// ActionService defines the interface for action processing.
type ActionService interface {
	ProcessAction(req entity.WebhookRequest, jobID string)
}

type actionService struct {
	systemExecutor system.Executor
}

// NewActionService creates a new action service with the given system executor.
func NewActionService(executor system.Executor) ActionService {
	return &actionService{
		systemExecutor: executor,
	}
}

func (s *actionService) ProcessAction(req entity.WebhookRequest, jobID string) {
	log.Printf("Starting job %s: action=%s", jobID, req.Action)

	switch req.Action {
	case entity.ActionReinit:
		s.executeCloudInit(jobID)
	case entity.ActionReboot:
		s.executeReboot(jobID)
	case entity.ActionUpdate:
		s.executeUpdate(jobID)
	default:
		log.Printf("Job %s: Unknown action '%s'", jobID, req.Action)
	}
}

func (s *actionService) executeCloudInit(jobID string) {
	log.Printf("Job %s: Executing cloud-init", jobID)

	if err := s.systemExecutor.RunCloudInit(); err != nil {
		log.Printf("Job %s: cloud-init failed: %v", jobID, err)
		return
	}

	log.Printf("Job %s: cloud-init completed successfully", jobID)
}

func (s *actionService) executeReboot(jobID string) {
	log.Printf("Job %s: Scheduling system reboot in 10 seconds", jobID)

	go func() {
		time.Sleep(getRebootDelay())
		if err := s.systemExecutor.Reboot(); err != nil {
			log.Printf("Job %s: reboot failed: %v", jobID, err)
		}
	}()
}

func (s *actionService) executeUpdate(jobID string) {
	log.Printf("Job %s: Executing system update", jobID)

	distro := s.systemExecutor.DetectDistribution()
	log.Printf("Job %s: Detected distribution: %s", jobID, distro)

	if err := s.systemExecutor.UpdateSystem(); err != nil {
		log.Printf("Job %s: system update failed: %v", jobID, err)
		return
	}

	log.Printf("Job %s: system update completed successfully", jobID)
}

// GenerateJobID generates a unique job identifier.
// Deprecated: Use security.GenerateJobID() for secure job ID generation.
func GenerateJobID() string {
	// This function is kept for backward compatibility
	// New code should use security.GenerateJobID()
	return fmt.Sprintf("job_%d_%d", time.Now().UnixNano(), time.Now().Unix()%1000)
}
