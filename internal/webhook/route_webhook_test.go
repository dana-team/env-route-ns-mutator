package webhook

import (
	"fmt"
	"testing"

	"github.com/dana-team/env-route-ns-mutator/internal/environment"

	. "github.com/onsi/gomega"
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	env1                 = "env1"
	env2                 = "env2"
	testNamespace        = "test-ns"
	clusterIngressDomain = "apps.ocp-test.os-test.com"
)

func TestRouteMutator(t *testing.T) {
	logger := ctrl.Log.WithName("webhook")

	environments := []string{env1, env2}

	tests := []struct {
		name          string
		namespace     string
		hostname      string
		customDomain  string
		defaultDomain bool
		nsLabels      map[string]string
		mutated       bool
	}{
		{name: "routeWithCustomNameCustomDomain", namespace: testNamespace, hostname: "test1", customDomain: "custom.com", defaultDomain: false, nsLabels: map[string]string{environment.Key: env1}, mutated: false},
		{name: "routeWithCustomNameNoDomain", namespace: testNamespace, hostname: "test3", customDomain: "", defaultDomain: false, nsLabels: map[string]string{environment.Key: env1}, mutated: false},
		{name: "routeWithCustomNameDefaultDomain", namespace: testNamespace, hostname: "test2", customDomain: "", defaultDomain: true, nsLabels: map[string]string{environment.Key: env1}, mutated: true},
		{name: "routeWithNoCustomNameNoDomain", namespace: testNamespace, hostname: "", customDomain: "", defaultDomain: false, nsLabels: map[string]string{environment.Key: env1}, mutated: true},
		{name: "routeWithoutLabels", namespace: testNamespace, hostname: "test5", customDomain: "", defaultDomain: true, nsLabels: map[string]string{}, mutated: false},
		{name: "routeWithBypassLabel", namespace: testNamespace, hostname: "test6", customDomain: "", defaultDomain: true, nsLabels: map[string]string{bypassLabel: "true", environment.Key: env1}, mutated: false},
		{name: "routeWithInvalidBypassLabel", namespace: testNamespace, hostname: "test7", customDomain: "", defaultDomain: true, nsLabels: map[string]string{bypassLabel: "false", environment.Key: env1}, mutated: true},
	}

	client := testclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			decoder := admission.NewDecoder(scheme.Scheme)
			rm := RouteMutator{Decoder: decoder, Client: client}

			routeHost := tc.hostname
			if len(tc.customDomain) > 0 {
				routeHost = fmt.Sprintf("%s.%s", tc.hostname, tc.customDomain)
			} else if tc.defaultDomain {
				routeHost = fmt.Sprintf("%s.%s", routeHost, clusterIngressDomain)
			}

			route := &routev1.Route{
				ObjectMeta: metav1.ObjectMeta{Name: tc.name, Namespace: tc.namespace},
				Spec:       routev1.RouteSpec{Host: routeHost},
			}

			rm.handleInner(logger, route, clusterIngressDomain, environments, tc.nsLabels)

			mutatedHost := ""
			if tc.mutated {
				switch {
				case len(routeHost) == 0:
					mutatedHost = fmt.Sprintf("%s-%s.%s-%s", tc.name, tc.namespace, tc.nsLabels[environment.Key], clusterIngressDomain)
				case tc.defaultDomain:
					mutatedHost = fmt.Sprintf("%s.%s-%s", tc.hostname, tc.nsLabels[environment.Key], clusterIngressDomain)
				default:
					mutatedHost = routeHost
				}
				g.Expect(route.Spec.Host).To(Equal(mutatedHost))
			} else {
				g.Expect(route.Spec.Host).To(Equal(routeHost))
			}
		})
	}
}
