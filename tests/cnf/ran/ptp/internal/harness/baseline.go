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
