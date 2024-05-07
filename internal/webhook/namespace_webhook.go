package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dana-team/env-route-ns-mutator/internal/environment"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// NamespaceMutator is the struct used to mutate Routes
type NamespaceMutator struct {
	Decoder admission.Decoder
	Client  client.Client
}

const DefaultSchedulerAnnotation = "scheduler.alpha.kubernetes.io/defaultTolerations"

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch

// +kubebuilder:webhook:path=/mutate-v1-namespace,mutating=true,failurePolicy=ignore,sideEffects=None,groups="",resources=namespaces,verbs=create;update,versions=v1,name=namespace.dana.io,admissionReviewVersions=v1;v1beta1

func (r *NamespaceMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithName("Namespace").WithValues("name", req.Name)
	logger.Info("webhook request received")

	namespace := corev1.Namespace{}
	if err := r.Decoder.Decode(req, &namespace); err != nil {
		logger.Error(err, "failed to decode namespace object")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	environments := environment.GetEnvironments()
	r.handleInner(logger, &namespace, environments)

	marshaledNamespace, err := json.Marshal(namespace)
	if err != nil {
		admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledNamespace)
}

// handleInner implements the main mutating logic. It modifies the labels of
// a Namespace based on environment data.
func (r *NamespaceMutator) handleInner(logger logr.Logger, namespace *corev1.Namespace, environments []string) {
	if value, ok := namespace.Annotations[DefaultSchedulerAnnotation]; ok {
		for _, env := range environments {
			toleration := fmt.Sprintf("[{\"operator\": \"Exists\", \"effect\": \"NoSchedule\", \"key\": %s}]", env)

			if value == toleration {
				labels := appendLabels(namespace.GetLabels(), map[string]string{environment.Key: env})
				namespace.SetLabels(labels)
				logger.Info("successfully updated labels")
			}
		}
	}
}

// appendLabels appends the received labels to the namespace.
func appendLabels(nsLabels, labels map[string]string) map[string]string {
	if len(nsLabels) == 0 {
		nsLabels = map[string]string{}
	}

	for key, value := range labels {
		nsLabels[key] = value
	}

	return nsLabels
}
