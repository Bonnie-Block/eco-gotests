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

// HoldoverPluginSettingsNominal are holdover thresholds applied during upstream-loss holdover tests.
// Values align with OCPBUGS-111642 QE guidance (maxInSpecOffset 40, localMaxHoldoverOffset 100,
// localHoldoverTimeout 300) and match XR8720t maxInSpecOffset; they replace the former WPC lab
// stress profile (360 / 14401|1800 / 14400). Tests patch these for the case duration and restore
// cluster defaults in cleanup.
var HoldoverPluginSettingsNominal = HoldoverPluginSettings{
	LocalHoldoverTimeout:   300,
	MaxInSpecOffset:        40,
	LocalMaxHoldoverOffSet: 100,
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
