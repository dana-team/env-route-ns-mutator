package utils

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/client"

	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	Env                = "environments"
	Key                = "environment"
	clusterIngressName = "cluster"
	bypassLabel        = "haproxy.router.dana.io/bypass-env-mutation"
)

// GetEnvironments retrieves environment data from a comma-separated environment variable.
// It returns a slice containing the environments.
func GetEnvironments() []string {
	environments := os.Getenv(Env)
	return strings.Split(environments, ",")
}

// GetClusterIngressDomain returns the ingress domain of an OpenShift cluster
func GetClusterIngressDomain(ctx context.Context, k8sClient client.Client) (string, error) {
	ingress := configv1.Ingress{}
	if err := k8sClient.Get(ctx, types.NamespacedName{Name: clusterIngressName}, &ingress); err != nil {
		return "", err
	}
	return ingress.Spec.Domain, nil
}

// CheckBypass checks if the namespace has the bypass mutation label.
func CheckBypass(labels map[string]string) bool {
	if val, ok := labels[bypassLabel]; ok && val == "true" {
		return true
	}

	return false
}

// AppendLabels appends the received labels to the namespace.
func AppendLabels(nsLabels, labels map[string]string) map[string]string {
	if len(nsLabels) == 0 {
		nsLabels = map[string]string{}
	}

	for key, value := range labels {
		nsLabels[key] = value
	}

	return nsLabels
}

// ModifyHostname modifies the hostname of the route/hostname based on the provided env.
func ModifyHostname(logger logr.Logger, objectName, objectNamespace, hostName, env, clusterIngress string) string {
	modifiedHostname := hostName
	switch {
	case len(hostName) == 0:
		modifiedHostname = fmt.Sprintf("%s-%s.%s-%s", objectName, objectNamespace, env, clusterIngress)
		logger.Info("Hostname is empty, modifying", "hostname", hostName)
	case strings.Contains(hostName, fmt.Sprintf("%s-%s", env, clusterIngress)):
		logger.Info("Hostname already includes environment, remains unchanged", "hostname", hostName)
	case strings.Contains(hostName, clusterIngress):
		environmentIngress := fmt.Sprintf("%s-%s", env, clusterIngress)
		modifiedHostname = strings.Replace(hostName, clusterIngress, environmentIngress, 1)
		logger.Info("Hostname includes cluster ingress, modifying", "hostname", hostName)
	default:
		logger.Info("Hostname is shortened, remains unchanged", "hostname", hostName)
	}
	return modifiedHostname
}
