package webhook

import (
	"fmt"
	"testing"

	networkingv1 "k8s.io/api/networking/v1"

	"github.com/dana-team/env-route-ns-mutator/internal/utils"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

const (
	testPath    = "/test"
	bypassLabel = "haproxy.router.dana.io/bypass-env-mutation"
)

func TestIngressMutator(t *testing.T) {
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
		{name: "ingressWithCustomNameCustomDomain", namespace: testNamespace, hostname: "test1", customDomain: "custom.com", defaultDomain: false, nsLabels: map[string]string{utils.Key: env1}, mutated: false},
		{name: "ingressWithCustomNameNoDomain", namespace: testNamespace, hostname: "test3", customDomain: "", defaultDomain: false, nsLabels: map[string]string{utils.Key: env1}, mutated: false},
		{name: "ingressWithCustomNameDefaultDomain", namespace: testNamespace, hostname: "test2", customDomain: "", defaultDomain: true, nsLabels: map[string]string{utils.Key: env1}, mutated: true},
		{name: "ingressWithNoCustomNameNoDomain", namespace: testNamespace, hostname: "", customDomain: "", defaultDomain: false, nsLabels: map[string]string{utils.Key: env1}, mutated: true},
		{name: "ingressWithoutLabels", namespace: testNamespace, hostname: "test5", customDomain: "", defaultDomain: true, nsLabels: map[string]string{}, mutated: false},
		{name: "ingressWithBypassLabel", namespace: testNamespace, hostname: "test6", customDomain: "", defaultDomain: true, nsLabels: map[string]string{bypassLabel: "true", utils.Key: env1}, mutated: false},
		{name: "ingressWithInvalidBypassLabel", namespace: testNamespace, hostname: "test7", customDomain: "", defaultDomain: true, nsLabels: map[string]string{bypassLabel: "false", utils.Key: env1}, mutated: true},
		{name: "ingressWithMutatedHostname", namespace: testNamespace, hostname: "test8", customDomain: fmt.Sprintf("%s-%s", env1, clusterIngressDomain), defaultDomain: false, nsLabels: map[string]string{utils.Key: env1}, mutated: false},
	}

	client := testclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			decoder := admission.NewDecoder(scheme.Scheme)
			rm := IngressMutator{Decoder: decoder, Client: client}

			ingressRuleHost := tc.hostname
			if len(tc.customDomain) > 0 {
				ingressRuleHost = fmt.Sprintf("%s.%s", tc.hostname, tc.customDomain)
			} else if tc.defaultDomain {
				ingressRuleHost = fmt.Sprintf("%s.%s", ingressRuleHost, clusterIngressDomain)
			}

			ingress := networkingv1.Ingress{
				ObjectMeta: metav1.ObjectMeta{Name: tc.name, Namespace: tc.namespace},
				Spec: networkingv1.IngressSpec{
					Rules: []networkingv1.IngressRule{
						{
							Host: ingressRuleHost,
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path: testPath,
										},
									},
								},
							},
						},
					},
				},
			}

			rm.handleInner(logger, &ingress, clusterIngressDomain, environments, tc.nsLabels)

			mutatedHost := ""
			if tc.mutated {
				switch {
				case len(ingressRuleHost) == 0:
					mutatedHost = fmt.Sprintf("%s-%s.%s-%s", tc.name, tc.namespace, tc.nsLabels[utils.Key], clusterIngressDomain)
				case tc.defaultDomain:
					mutatedHost = fmt.Sprintf("%s.%s-%s", tc.hostname, tc.nsLabels[utils.Key], clusterIngressDomain)
				default:
					mutatedHost = ingressRuleHost
				}
				g.Expect(ingress.Spec.Rules[0].Host).To(Equal(mutatedHost))
			} else {
				g.Expect(ingress.Spec.Rules[0].Host).To(Equal(ingressRuleHost))
			}
		})
	}
}
