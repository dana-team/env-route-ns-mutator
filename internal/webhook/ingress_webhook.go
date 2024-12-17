package webhook

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-logr/logr"

	"github.com/dana-team/env-route-ns-mutator/internal/utils"

	networkingv1 "k8s.io/api/networking/v1"

	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// IngressMutator is the struct used to mutate ingresses.
type IngressMutator struct {
	Decoder admission.Decoder
	Client  client.Client
}

// +kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch

// +kubebuilder:webhook:path=/mutate-v1-ingress,mutating=true,failurePolicy=ignore,sideEffects=None,groups=networking.k8s.io,resources=ingresses,verbs=create,versions=v1,name=ingress.dana.io,admissionReviewVersions=v1;v1beta1

func (r *IngressMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithName("Ingress").WithValues("name", req.Name)
	logger.Info("webhook request received")

	ingress := networkingv1.Ingress{}
	if err := r.Decoder.Decode(req, &ingress); err != nil {
		logger.Error(err, "failed to decode ingress object")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	namespace := corev1.Namespace{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: req.Namespace}, &namespace); err != nil {
		logger.Error(err, "failed to get namespace object")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	clusterIngress, err := utils.GetClusterIngressDomain(ctx, r.Client)
	if err != nil {
		logger.Error(err, "failed to get cluster ingress")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	environments := utils.GetEnvironments()
	r.handleInner(logger, &ingress, clusterIngress, environments, namespace.ObjectMeta.Labels)

	marshaledIngress, err := json.Marshal(ingress)
	if err != nil {
		admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledIngress)
}

func (r *IngressMutator) handleInner(logger logr.Logger, ingress *networkingv1.Ingress, clusterIngress string, environments []string, namespaceLabels map[string]string) {
	if utils.CheckBypass(namespaceLabels) {
		logger.Info("Bypassing mutation")
		return
	}

	for _, env := range environments {
		if namespaceLabels[utils.Key] == env {
			for i, rule := range ingress.Spec.Rules {
				ruleHost := utils.ModifyHostname(logger, ingress.Name, ingress.Namespace, rule.Host, env, clusterIngress)
				ingress.Spec.Rules[i].Host = ruleHost
			}
			break

		}
	}
}
