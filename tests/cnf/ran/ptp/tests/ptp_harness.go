package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/daemonlogs"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/harness"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/metrics"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/tsparams"
	"k8s.io/klog/v2"
)

const clocksNotLockedSkipReason = "PTP clocks are not locked and suite baseline recovery did not restore LOCKED state; " +
	"the cluster may be degraded by an earlier spec (for example OCP-73095, see OCPBUGS-104531). " +
	"Skipping remaining assertions for this spec so later specs can run or skip explicitly (CNF-26726)."

// ensureClocksLockedBeforeSpec verifies clock lock with a short timeout, restores the suite PTP baseline when needed,
// and skips the current spec when the cluster cannot be returned to a locked steady state.
func ensureClocksLockedBeforeSpec(prometheusAPI prometheusv1.API) {
	By("ensuring clocks are locked before testing")

	err := metrics.EnsureClocksAreLockedForSpecSetup(prometheusAPI)
	if err == nil {
		return
	}

	klog.V(tsparams.LogLevel).Infof("PTP clocks not locked before spec (%v); attempting suite baseline recovery", err)

	By("restoring suite PTP baseline to recover clock lock state")

	if err := recoverClockLockFromSuiteBaseline(prometheusAPI); err != nil {
		Skip(fmt.Sprintf("%s: %v", clocksNotLockedSkipReason, err))
	}
}

// ensureClocksLockedAfterSpec attempts to leave the cluster locked after a spec without failing the spec on cleanup.
// A failed cleanup is logged; the next spec's BeforeEach will try baseline recovery again.
func ensureClocksLockedAfterSpec(prometheusAPI prometheusv1.API) {
	By("ensuring clocks are locked after testing")

	err := metrics.EnsureClocksAreLockedForSpecSetup(prometheusAPI)
	if err == nil {
		return
	}

	klog.V(tsparams.LogLevel).Infof("PTP clocks not locked after spec (%v); attempting suite baseline recovery", err)

	By("restoring suite PTP baseline after spec")

	if err := recoverClockLockFromSuiteBaseline(prometheusAPI); err != nil {
		klog.Infof("PTP clocks still not locked after spec cleanup and baseline recovery: %v", err)
	}
}

func recoverClockLockFromSuiteBaseline(prometheusAPI prometheusv1.API) error {
	startTime := time.Now()

	changedProfiles, err := harness.RestoreBaseline(RANConfig.Spoke1APIClient)
	if err != nil {
		return err
	}

	if len(changedProfiles) > 0 {
		err = daemonlogs.WaitForProfileLoadOnPTPNodes(RANConfig.Spoke1APIClient,
			daemonlogs.WithStartTime(startTime),
			daemonlogs.WithTimeout(5*time.Minute))
		if err != nil {
			klog.V(tsparams.LogLevel).Infof("Failed to wait for profile load during baseline recovery: %v", err)
		}
	}

	err = metrics.EnsureClocksAreLocked(prometheusAPI)
	if err != nil {
		return err
	}

	return nil
}
