package version

// The version package imports cluster (via version.go), which pulls inittools. For local unit tests of
// IsVersionStringInRange / clusterExtensionBundleVersion, run:
// UNIT_TEST=true go test ./tests/cnf/ran/internal/version/... -run 'TestIsVersionStringInRange|TestClusterExtensionBundleVersion'

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

//nolint:funlen
func TestIsVersionStringInRange(t *testing.T) {
	testCases := []struct {
		version          string
		minimum          string
		maximum          string
		expectedResult   bool
		wantErrSubstring string // if non-empty, err must contain this substring; if empty, err must be nil
	}{
		{
			version:        "4.16.0",
			minimum:        "4.10.0-0",
			maximum:        "",
			expectedResult: true,
		},
		{
			version:        "4.16.0",
			minimum:        "",
			maximum:        "4.21.0-0",
			expectedResult: true,
		},
		{
			version:        "4.16.0",
			minimum:        "4.10.0-0",
			maximum:        "4.21.0-0",
			expectedResult: true,
		},
		{
			version:        "4.16.0",
			minimum:        "4.20.0-0",
			maximum:        "",
			expectedResult: false,
		},
		{
			version:        "4.16.0",
			minimum:        "",
			maximum:        "4.11.0-0",
			expectedResult: false,
		},
		{
			version:        "4.16.0",
			minimum:        "4.10.0-0",
			maximum:        "4.16.0-0",
			expectedResult: false,
		},
		{
			version:        "4.16.0",
			minimum:        "4.0.0-0",
			maximum:        "5.1.0-0",
			expectedResult: true,
		},
		{
			version:        "4.16.0",
			minimum:        "3.0.0-0",
			maximum:        "4.1.0-0",
			expectedResult: false,
		},
		{
			version:          "4.16.0",
			minimum:          "invalid minimum",
			maximum:          "",
			expectedResult:   false,
			wantErrSubstring: "invalid minimum provided: 'invalid minimum'",
		},
		{
			version:          "4.16.0",
			minimum:          "",
			maximum:          "invalid maximum",
			expectedResult:   false,
			wantErrSubstring: "invalid maximum provided: 'invalid maximum'",
		},
		{
			version:        "",
			minimum:        "3.0.0-0",
			maximum:        "4.1.0-0",
			expectedResult: false,
		},
		{
			version:        "",
			minimum:        "3.0.0-0",
			maximum:        "",
			expectedResult: true,
		},
		{
			version:        "4.20.0-20251212.151256",
			minimum:        "4.20.0",
			maximum:        "",
			expectedResult: false,
		},
		{
			version:        "4.20.0-20251212.151256",
			minimum:        "4.20.0-0",
			maximum:        "",
			expectedResult: true,
		},
		{
			version:        "v4.16.5",
			minimum:        "4.16.0-0",
			maximum:        "",
			expectedResult: true,
		},
		// Explicit exclusive maximum: v < 4.18.0-0 (4.18.0 release is out).
		{
			version:        "4.17.5",
			minimum:        "",
			maximum:        "4.18.0-0",
			expectedResult: true,
		},
		{
			version:        "4.18.0",
			minimum:        "",
			maximum:        "4.18.0-0",
			expectedResult: false,
		},
		{
			version:        "4.18.5",
			minimum:        "",
			maximum:        "4.18.0-0",
			expectedResult: false,
		},
	}

	for _, testCase := range testCases {
		result, err := IsVersionStringInRange(testCase.version, testCase.minimum, testCase.maximum)

		assert.Equal(t, testCase.expectedResult, result)

		if testCase.wantErrSubstring != "" {
			assert.Error(t, err)
			assert.ErrorContains(t, err, testCase.wantErrSubstring)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestClusterExtensionBundleVersion(t *testing.T) {
	testCases := []struct {
		name              string
		obj               *unstructured.Unstructured
		operatorName      string
		operatorNamespace string
		wantVersion       string
		wantOK            bool
	}{
		{
			name: "match by ClusterExtension name",
			obj: unstructuredFromMap(map[string]interface{}{
				"metadata": map[string]interface{}{"name": "ptp-operator"},
				"spec": map[string]interface{}{
					"namespace": "openshift-ptp",
					"source": map[string]interface{}{
						"catalog": map[string]interface{}{
							"packageName": "ptp-operator",
						},
					},
				},
				"status": map[string]interface{}{
					"install": map[string]interface{}{
						"bundle": map[string]interface{}{
							"name":    "ptp-operator.v5.0.0-202607290430",
							"version": "5.0.0-202607290430",
						},
					},
				},
			}),
			operatorName:      "ptp-operator",
			operatorNamespace: "openshift-ptp",
			wantVersion:       "5.0.0-202607290430",
			wantOK:            true,
		},
		{
			name: "namespace mismatch",
			obj: unstructuredFromMap(map[string]interface{}{
				"metadata": map[string]interface{}{"name": "ptp-operator"},
				"spec": map[string]interface{}{
					"namespace": "other-ns",
					"source": map[string]interface{}{
						"catalog": map[string]interface{}{
							"packageName": "ptp-operator",
						},
					},
				},
				"status": map[string]interface{}{
					"install": map[string]interface{}{
						"bundle": map[string]interface{}{
							"name":    "ptp-operator.v5.0.0-202607290430",
							"version": "5.0.0-202607290430",
						},
					},
				},
			}),
			operatorName:      "ptp-operator",
			operatorNamespace: "openshift-ptp",
			wantVersion:       "",
			wantOK:            false,
		},
		{
			name: "missing install version",
			obj: unstructuredFromMap(map[string]interface{}{
				"metadata": map[string]interface{}{"name": "ptp-operator"},
				"spec": map[string]interface{}{
					"namespace": "openshift-ptp",
					"source": map[string]interface{}{
						"catalog": map[string]interface{}{
							"packageName": "ptp-operator",
						},
					},
				},
			}),
			operatorName:      "ptp-operator",
			operatorNamespace: "openshift-ptp",
			wantVersion:       "",
			wantOK:            false,
		},
		{
			name:              "nil object",
			obj:               nil,
			operatorName:      "ptp-operator",
			operatorNamespace: "openshift-ptp",
			wantVersion:       "",
			wantOK:            false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			gotVersion, gotOK := clusterExtensionBundleVersion(
				testCase.obj, testCase.operatorName, testCase.operatorNamespace)

			assert.Equal(t, testCase.wantOK, gotOK)
			assert.Equal(t, testCase.wantVersion, gotVersion)
		})
	}
}

func unstructuredFromMap(content map[string]interface{}) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: content}
}
