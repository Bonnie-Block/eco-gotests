package tests

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/ginkgo/v2/types"
	. "github.com/onsi/gomega"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	. "github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/raninittools"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/daemonlogs"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/harness"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/metrics"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/profiles"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/ptp/internal/tsparams"
	"k8s.io/klog/v2"
)

const clocksNotLockedFailReason = "PTP clocks are not locked and suite baseline recovery did not restore LOCKED state; " +
	"the cluster may be degraded by an earlier spec (for example OCP-73095, see OCPBUGS-104531). " +
	"This is a harness isolation failure, not an assertion of the spec under test (CNF-26726)."

// ensureClocksLockedBeforeSpec verifies clock lock with a short timeout, restores the suite PTP baseline when needed,
// and fails the current spec when the cluster cannot be returned to a locked steady state.
func ensureClocksLockedBeforeSpec(prometheusAPI prometheusv1.API) {
	GinkgoHelper()

	By("ensuring clocks are locked before testing")

	err := metrics.EnsureClocksAreLockedForSpecSetup(prometheusAPI)
	if err == nil {
		return
	}

	failIfClocksRemainUnlocked(prometheusAPI, err, "before spec")
}

// ensureClocksLockedAfterSpec restores lock after a spec when needed. Leftover FREERUN after baseline
// recovery fails the spec so a passing It cannot hide cluster pollution. Skipped specs do not fail.
func ensureClocksLockedAfterSpec(prometheusAPI prometheusv1.API) {
	GinkgoHelper()

	if CurrentSpecReport().State == types.SpecStateSkipped {
		return
	}

	By("ensuring clocks are locked after testing")

	err := metrics.EnsureClocksAreLockedForSpecSetup(prometheusAPI)
	if err == nil {
		return
	}

	failIfClocksRemainUnlocked(prometheusAPI, err, "after spec")
}

func failIfClocksRemainUnlocked(prometheusAPI prometheusv1.API, firstErr error, when string) {
	GinkgoHelper()

	if alreadyFailed, recoveryErr := harness.RecoveryAlreadyFailed(); alreadyFailed {
		Fail(fmt.Sprintf("%s (%s); suite PTP baseline recovery already failed: %v",
			clocksNotLockedFailReason, when, recoveryErr))
	}

	klog.V(tsparams.LogLevel).Infof(
		"PTP clocks not locked %s (%v); attempting suite baseline recovery", when, firstErr)

	By("restoring suite PTP baseline to recover clock lock state")

	err := recoverClockLockFromSuiteBaseline(prometheusAPI)
	if err != nil {
		harness.MarkRecoveryFailed(err)
		Fail(fmt.Sprintf("%s (%s): %v", clocksNotLockedFailReason, when, err))
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

// skipIfNoNodeHasProfileTypes skips the spec when no node has any of the given profile types.
// Call this in BeforeEach before ensureClocksLockedBeforeSpec so HA2BC-incompatible specs Skip
// for topology instead of failing a lock check (CNF-26726, CNF-26740).
func skipIfNoNodeHasProfileTypes(skipReason string, profileTypes ...profiles.PtpProfileType) {
	GinkgoHelper()

	By("checking whether the cluster has profiles required for this spec")

	nodeInfoMap, err := profiles.GetNodeInfoMap(RANConfig.Spoke1APIClient)
	Expect(err).ToNot(HaveOccurred(), "Failed to get node info map")

	for _, nodeInfo := range nodeInfoMap {
		for _, profileType := range profileTypes {
			if nodeInfo.Counts[profileType] > 0 {
				return
			}
		}
	}

	Skip(skipReason)
}

// skipIfNoNodeWithoutProfileType skips the spec when every node has the given profile type.
func skipIfNoNodeWithoutProfileType(skipReason string, profileType profiles.PtpProfileType) {
	GinkgoHelper()

	By("checking whether any node lacks the excluded profile type")

	nodeInfoMap, err := profiles.GetNodeInfoMap(RANConfig.Spoke1APIClient)
	Expect(err).ToNot(HaveOccurred(), "Failed to get node info map")

	for _, nodeInfo := range nodeInfoMap {
		if nodeInfo.Counts[profileType] == 0 {
			return
		}
	}

	Skip(skipReason)
}
