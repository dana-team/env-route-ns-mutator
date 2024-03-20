package webhook

import (
	"fmt"
	"testing"

	"github.com/dana-team/env-route-ns-mutator/internal/environment"
	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	testclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func TestNamespaceMutator(t *testing.T) {
	logger := ctrl.Log.WithName("webhook")

	environments := []string{env1, env2}

	tests := []struct {
		name      string
		namespace string
		env       string
		mutated   bool
	}{
		{name: "namespaceWithEnvironmentToleration", namespace: testNamespace, env: env1, mutated: true},
		{name: "namespaceWithoutEnvironmentToleration", namespace: testNamespace, env: "no-in-env-list", mutated: false},
	}

	client := testclient.NewClientBuilder().WithScheme(scheme.Scheme).Build()
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)

			decoder := admission.NewDecoder(scheme.Scheme)
			rm := NamespaceMutator{Decoder: decoder, Client: client}

			namespace := &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:        tc.name,
					Namespace:   tc.namespace,
					Annotations: map[string]string{DefaultSchedulerAnnotation: fmt.Sprintf("[{\"operator\": \"Exists\", \"effect\": \"NoSchedule\", \"key\": %s}]", tc.env)},
				},
			}

			rm.handleInner(logger, namespace, environments)

			if tc.mutated {
				g.Expect(namespace.GetLabels()[environment.Key]).To(Equal(tc.env))
			} else {
				g.Expect(namespace.GetLabels()[environment.Key]).To(BeEmpty())
			}
		})
	}
}
