//go:build unit_test

package harness

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryState(t *testing.T) {
	ResetRecoveryState()
	t.Cleanup(ResetRecoveryState)

	failed, err := RecoveryAlreadyFailed()
	assert.False(t, failed)
	assert.NoError(t, err)

	marked := errors.New("baseline restore left clocks FREERUN")
	MarkRecoveryFailed(marked)

	failed, err = RecoveryAlreadyFailed()
	assert.True(t, failed)
	require.ErrorIs(t, err, marked)

	ResetRecoveryState()

	failed, err = RecoveryAlreadyFailed()
	assert.False(t, failed)
	assert.NoError(t, err)
}

func TestRestoreBaselineWithoutCapture(t *testing.T) {
	baselineMu.Lock()
	baselineConfigs = nil
	baselineMu.Unlock()
	t.Cleanup(func() {
		baselineMu.Lock()
		baselineConfigs = nil
		recoveryFailed = false
		recoveryErr = nil
		baselineMu.Unlock()
	})

	changed, err := RestoreBaseline(nil)
	assert.Nil(t, changed)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "suite PTP baseline was not captured")
}
