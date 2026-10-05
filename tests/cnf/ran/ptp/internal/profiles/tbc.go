package profiles

import (
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/iface"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/metrics"
)

// HoldoverTestData groups the per-node test context that is discovered once in BeforeEach and shared
// by all test cases within a Context block.
type HoldoverTestData struct {
	PrometheusAPI  prometheusv1.API
	NodeName       string
	ProfileInfo    *ProfileInfo
	UpstreamIfaces []iface.Iface
	ConfigFile     string
}

// HoldoverExpectedClockClasses groups the expected clock class values for each holdover state.
type HoldoverExpectedClockClasses struct {
	Locked            metrics.PtpClockClass
	HoldoverInSpec    metrics.PtpClockClass
	HoldoverOutOfSpec metrics.PtpClockClass
	Freerun           metrics.PtpClockClass
}

// Holdover presets for upstream-loss tests (OCPBUGS-111642 QE-aligned timing, not WPC stress).
// Tests patch one preset per case and restore cluster defaults in cleanup.

// HoldoverPluginSettingsNoOutOfSpec: localMaxHoldoverOffSet < maxInSpecOffset so FREERUN is
// reached before holdover-out-of-spec (83297/83298, 88274/88275).
var HoldoverPluginSettingsNoOutOfSpec = HoldoverPluginSettings{
	LocalHoldoverTimeout:   300,
	MaxInSpecOffset:        200,
	LocalMaxHoldoverOffSet: 100,
}

// HoldoverPluginSettingsWithOutOfSpec: maxInSpecOffset < localMaxHoldoverOffSet so
// holdover-out-of-spec occurs before FREERUN (83299/83300, 88276/88277).
var HoldoverPluginSettingsWithOutOfSpec = HoldoverPluginSettings{
	LocalHoldoverTimeout:   300,
	MaxInSpecOffset:        40,
	LocalMaxHoldoverOffSet: 1500,
}

// TBCClockClasses returns the standard clock class values for T-BC tests.
func TBCClockClasses() HoldoverExpectedClockClasses {
	return HoldoverExpectedClockClasses{
		Locked:            metrics.ClockClass6,
		HoldoverInSpec:    metrics.ClockClass135,
		HoldoverOutOfSpec: metrics.ClockClass165,
		Freerun:           metrics.ClockClass248,
	}
}
