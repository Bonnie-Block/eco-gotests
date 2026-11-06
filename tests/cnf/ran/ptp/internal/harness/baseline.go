package harness

import (
	"fmt"
	"sync"

	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/ptp"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/profiles"
)

var (
	baselineMu      sync.RWMutex
	baselineConfigs []*ptp.PtpConfigBuilder
	recoveryFailed  bool
	recoveryErr     error
)

// CaptureBaseline stores a snapshot of all PtpConfigs in the cluster for suite-level recovery.
// It should be called once after the cluster is verified healthy at suite start.
func CaptureBaseline(client *clients.Settings) error {
	saved, err := profiles.SavePtpConfigs(client)
	if err != nil {
		return fmt.Errorf("failed to capture suite PTP baseline: %w", err)
	}

	baselineMu.Lock()
	baselineConfigs = saved
	recoveryFailed = false
	recoveryErr = nil
	baselineMu.Unlock()

	return nil
}

// RestoreBaseline reapplies the suite-start PtpConfigs to the cluster.
func RestoreBaseline(client *clients.Settings) ([]*profiles.ProfileReference, error) {
	baselineMu.RLock()
	saved := baselineConfigs
	baselineMu.RUnlock()

	if len(saved) == 0 {
		return nil, fmt.Errorf("suite PTP baseline was not captured")
	}

	changed, err := profiles.RestorePtpConfigs(client, saved)
	if err != nil {
		return nil, fmt.Errorf("failed to restore suite PTP baseline: %w", err)
	}

	return changed, nil
}

// RecoveryAlreadyFailed reports whether suite baseline recovery already failed to restore LOCKED
// clocks in this suite. When true, later specs must not wait another five minutes on the same
// FREERUN state (CNF-26726).
func RecoveryAlreadyFailed() (bool, error) {
	baselineMu.RLock()
	defer baselineMu.RUnlock()

	return recoveryFailed, recoveryErr
}

// MarkRecoveryFailed records that suite baseline recovery did not restore LOCKED clocks.
func MarkRecoveryFailed(err error) {
	baselineMu.Lock()
	recoveryFailed = true
	recoveryErr = err
	baselineMu.Unlock()
}

// ResetRecoveryState clears the suite recovery-failure flag. Tests use this to isolate cases;
// CaptureBaseline also clears it at suite start.
func ResetRecoveryState() {
	baselineMu.Lock()
	recoveryFailed = false
	recoveryErr = nil
	baselineMu.Unlock()
}
