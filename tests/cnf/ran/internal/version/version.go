package version

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
	configv1 "github.com/openshift/api/config/v1"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/argocd"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/clients"
	"github.com/rh-ecosystem-edge/eco-goinfra/pkg/olm"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/cnf/ran/internal/ranparam"
	"github.com/rh-ecosystem-edge/eco-gotests/tests/internal/cluster"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
)

var clusterExtensionGVR = schema.GroupVersionResource{
	Group:    "olm.operatorframework.io",
	Version:  "v1",
	Resource: "clusterextensions",
}

// GetOCPVersion uses the cluster version on a given cluster to find the latest OCP version, returning the desired
// version if the latest version could not be found.
func GetOCPVersion(client *clients.Settings) (string, error) {
	clusterVersion, err := cluster.GetOCPClusterVersion(client)
	if err != nil {
		return "", err
	}

	// Workaround for an issue in eco-goinfra where builder.Object is nil even when Pull returns a nil error.
	if clusterVersion.Object == nil {
		return "", fmt.Errorf("failed to get ClusterVersion object")
	}

	histories := clusterVersion.Object.Status.History
	for i := len(histories) - 1; i >= 0; i-- {
		if histories[i].State == configv1.CompletedUpdate {
			return histories[i].Version, nil
		}
	}

	klog.V(ranparam.LogLevel).Info("No completed cluster version found in history, returning desired version")

	return clusterVersion.Object.Status.Desired.Version, nil
}

// GetClusterName extracts the cluster name from provided kubeconfig, assuming there's one cluster in the kubeconfig.
func GetClusterName(kubeconfigPath string) (string, error) {
	rawConfig, _ := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: "",
		}).RawConfig()

	for _, cluster := range rawConfig.Clusters {
		// Get a cluster name by parsing it from the server hostname. Expects the url to start with
		// `https://api.cluster-name.` so splitting by `.` gives the cluster name.
		splits := strings.Split(cluster.Server, ".")
		clusterName := splits[1]

		klog.V(ranparam.LogLevel).Infof("cluster name %s found for kubeconfig at %s", clusterName, kubeconfigPath)

		return clusterName, nil
	}

	return "", fmt.Errorf("could not get cluster name for kubeconfig at %s", kubeconfigPath)
}

// GetOperatorVersion returns the operator version, preferring an OLMv0 ClusterServiceVersion and falling back to an
// OLMv1 ClusterExtension (status.install.bundle.version). This is required for operators installed via ClusterExtension
// where no CSV exists in the operator namespace.
func GetOperatorVersion(client *clients.Settings, operatorName, operatorNamespace string) (string, error) {
	version, csvErr := GetOperatorVersionFromCsv(client, operatorName, operatorNamespace)
	if csvErr == nil && version != "" {
		return version, nil
	}

	klog.V(ranparam.LogLevel).Infof(
		"CSV version lookup failed for operator %s in namespace %s (%v); trying ClusterExtension",
		operatorName, operatorNamespace, csvErr)

	version, ceErr := GetOperatorVersionFromClusterExtension(client, operatorName, operatorNamespace)
	if ceErr == nil && version != "" {
		return version, nil
	}

	if csvErr != nil && ceErr != nil {
		return "", fmt.Errorf(
			"could not find version for operator %s: CSV lookup: %v; ClusterExtension lookup: %w",
			operatorName, csvErr, ceErr)
	}

	if ceErr != nil {
		return "", ceErr
	}

	return "", csvErr
}

// GetOperatorVersionFromCsv returns operator version from a ClusterServiceVersion in operatorNamespace.
func GetOperatorVersionFromCsv(client *clients.Settings, operatorName, operatorNamespace string) (string, error) {
	csv, err := olm.ListClusterServiceVersion(client, operatorNamespace)
	if err != nil {
		return "", err
	}

	for _, csv := range csv {
		if strings.Contains(csv.Object.Name, operatorName) {
			return csv.Object.Spec.Version.String(), nil
		}
	}

	return "", fmt.Errorf("could not find version for operator %s in namespace %s", operatorName, operatorNamespace)
}

// GetOperatorVersionFromClusterExtension returns operator version from an OLMv1 ClusterExtension whose name, package,
// or installed bundle matches operatorName. When operatorNamespace is non-empty, ClusterExtensions installed into a
// different namespace are skipped.
func GetOperatorVersionFromClusterExtension(
	client *clients.Settings, operatorName, operatorNamespace string) (string, error) {
	if client == nil || client.Interface == nil {
		return "", fmt.Errorf("nil client while looking up ClusterExtension for operator %s", operatorName)
	}

	obj, err := client.Resource(clusterExtensionGVR).Get(context.TODO(), operatorName, metav1.GetOptions{})
	if err == nil {
		if version, ok := clusterExtensionBundleVersion(obj, operatorName, operatorNamespace); ok {
			return version, nil
		}
	}

	list, err := client.Resource(clusterExtensionGVR).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to list ClusterExtensions for operator %s: %w", operatorName, err)
	}

	for i := range list.Items {
		if version, ok := clusterExtensionBundleVersion(&list.Items[i], operatorName, operatorNamespace); ok {
			return version, nil
		}
	}

	return "", fmt.Errorf("could not find ClusterExtension version for operator %s", operatorName)
}

// clusterExtensionBundleVersion returns status.install.bundle.version when the ClusterExtension matches operatorName
// (and optional install namespace).
func clusterExtensionBundleVersion(
	obj *unstructured.Unstructured, operatorName, operatorNamespace string) (string, bool) {
	if obj == nil {
		return "", false
	}

	name := obj.GetName()
	packageName, _, _ := unstructured.NestedString(obj.Object, "spec", "source", "catalog", "packageName")
	installNamespace, _, _ := unstructured.NestedString(obj.Object, "spec", "namespace")
	bundleName, _, _ := unstructured.NestedString(obj.Object, "status", "install", "bundle", "name")
	version, _, _ := unstructured.NestedString(obj.Object, "status", "install", "bundle", "version")

	if version == "" {
		return "", false
	}

	matches := name == operatorName ||
		strings.Contains(name, operatorName) ||
		packageName == operatorName ||
		strings.Contains(packageName, operatorName) ||
		strings.Contains(bundleName, operatorName)
	if !matches {
		return "", false
	}

	if operatorNamespace != "" && installNamespace != "" && installNamespace != operatorNamespace {
		return "", false
	}

	return version, true
}

// GetZTPVersionFromArgoCd is used to fetch the version of the ztp-site-generate init container.
func GetZTPVersionFromArgoCd(client *clients.Settings, name, namespace string) (string, error) {
	containerImage, err := GetZTPSiteGenerateImage(client)
	if err != nil {
		return "", err
	}

	colonSplit := strings.Split(containerImage, ":")
	ztpVersion := colonSplit[len(colonSplit)-1]

	if ztpVersion == "latest" {
		klog.V(ranparam.LogLevel).Info("ztp-site-generate version tag was 'latest', returning empty version")

		return "", nil
	}

	// The format here will be like vX.Y.Z so we need to remove the v at the start.
	return ztpVersion[1:], nil
}

// GetZTPSiteGenerateImage returns the image used for the ztp-site-generate init container. It takes this from the Argo
// CD resource.
func GetZTPSiteGenerateImage(client *clients.Settings) (string, error) {
	gitops, err := argocd.Pull(client, ranparam.OpenshiftGitOpsNamespace, ranparam.OpenshiftGitOpsNamespace)
	if err != nil {
		return "", err
	}

	for _, container := range gitops.Definition.Spec.Repo.InitContainers {
		// Match both the `ztp-site-generator` and `ztp-site-generate` images since which one matches is version
		// dependent.
		if strings.Contains(container.Image, "ztp-site-gen") {
			return container.Image, nil
		}
	}

	return "", errors.New("unable to identify ZTP site generate image")
}

// IsVersionStringInRange reports whether version satisfies minimum <= version < maximumUpper using SemVer 2.0
// (github.com/Masterminds/semver/v3).
//
// minimum: empty means no lower bound. For a lower bound that must include pre-releases of X.Y.0 (e.g. OCP
// 4.20.0-20251212...), pass X.Y.0-0 — the lowest pre-release of X.Y.0. Plain X.Y.0 excludes pre-releases of X.Y.0.
//
// maximum: empty means no upper bound. Otherwise maximum is exclusive: the interval is minimum <= v < maximum
// (half-open). Non-empty bounds use the same rules as version: trim an optional leading v, then parse with
// github.com/Masterminds/semver/v3. Callers must pass semver-compatible strings (e.g. lower bound "4.16.0-0",
// exclusive upper "4.21.0-0").
//
// If version is not a valid semver string and maximum is empty, the function returns (true, nil) for compatibility
// with legacy call sites (e.g. empty or non-semver operator tags with no upper bound). Callers that require a parsed
// version should validate the string separately or pass a non-empty maximum so the result is (false, nil).
func IsVersionStringInRange(version, minimum, maximum string) (bool, error) {
	var minV *semver.Version

	if minimum != "" {
		parsed, err := semver.NewVersion(trimSemverVPrefix(minimum))
		if err != nil {
			return false, fmt.Errorf("invalid minimum provided: '%s'", minimum)
		}

		minV = parsed
	}

	var maxExclusive *semver.Version

	if maximum != "" {
		parsed, err := semver.NewVersion(trimSemverVPrefix(maximum))
		if err != nil {
			return false, fmt.Errorf("invalid maximum provided: '%s'", maximum)
		}

		maxExclusive = parsed
	}

	parsedVersion, err := semver.NewVersion(trimSemverVPrefix(version))
	if err != nil {
		if maximum == "" {
			klog.V(ranparam.LogLevel).Infof(
				"IsVersionStringInRange: unparsable version %q, returning (true, nil) for legacy no-upper-bound behavior: %v",
				version, err)

			return true, nil
		}

		return false, nil
	}

	if minV != nil && parsedVersion.LessThan(minV) {
		return false, nil
	}

	if maxExclusive != nil && !parsedVersion.LessThan(maxExclusive) {
		return false, nil
	}

	return true, nil
}

func trimSemverVPrefix(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "v")
}
