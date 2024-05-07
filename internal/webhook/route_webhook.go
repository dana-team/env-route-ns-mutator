package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/dana-team/env-route-ns-mutator/internal/environment"
	"github.com/go-logr/logr"
	configv1 "github.com/openshift/api/config/v1"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// RouteMutator is the struct used to mutate Routes
type RouteMutator struct {
	Decoder admission.Decoder
	Client  client.Client
}

const clusterIngressName = "cluster"

// +kubebuilder:rbac:groups="route.openshift.io",resources=routes,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups="config.openshift.io",resources=ingresses,verbs=get;list;watch

// +kubebuilder:webhook:path=/mutate-v1-route,mutating=true,failurePolicy=ignore,sideEffects=None,groups=route.openshift.io,resources=routes,verbs=create;update,versions=v1,name=route.dana.io,admissionReviewVersions=v1;v1beta1

func (r *RouteMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	logger := log.FromContext(ctx).WithName("Route").WithValues("name", req.Name)
	logger.Info("webhook request received")

	route := routev1.Route{}
	if err := r.Decoder.Decode(req, &route); err != nil {
		logger.Error(err, "failed to decode route object")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	namespace := corev1.Namespace{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: req.Namespace}, &namespace); err != nil {
		logger.Error(err, "failed to get namespace object")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	clusterIngress, err := r.getClusterIngressDomain(ctx)
	if err != nil {
		logger.Error(err, "failed to get cluster ingress")
		return admission.Errored(http.StatusInternalServerError, err)
	}

	environments := environment.GetEnvironments()
	r.handleInner(logger, &route, clusterIngress, environments, namespace.ObjectMeta.Labels)

	marshaledRoute, err := json.Marshal(route)
	if err != nil {
		admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledRoute)
}

// handleInner implements the main mutating logic. It modifies the host of an OpenShift Route
// based on environment data and cluster ingress information.
func (r *RouteMutator) handleInner(logger logr.Logger, route *routev1.Route, clusterIngress string, environments []string, labels map[string]string) {
	for _, env := range environments {
		if labels[environment.Key] == env {
			routeHost := route.Spec.Host
			switch {
			case len(routeHost) == 0:
				routeHost = fmt.Sprintf("%s-%s.%s-%s", route.Name, route.Namespace, env, clusterIngress)
				logger.Info("Route hostname is empty, modifying", "routeHost", routeHost)
			case strings.Contains(routeHost, clusterIngress):
				environmentIngress := fmt.Sprintf("%s-%s", env, clusterIngress)
				routeHost = strings.Replace(routeHost, clusterIngress, environmentIngress, 1)
				logger.Info("Route hostname includes cluster ingress, modifying", "routeHost", routeHost)
			default:
				logger.Info("Route hostname is shortened, remains unchanged", "routeHost", routeHost)
			}

			route.Spec.Host = routeHost
			break
		}
	}
}

// getClusterIngressDomain returns the ingress domain of an OpenShift cluster
func (r *RouteMutator) getClusterIngressDomain(ctx context.Context) (string, error) {
	ingress := configv1.Ingress{}
	if err := r.Client.Get(ctx, types.NamespacedName{Name: clusterIngressName}, &ingress); err != nil {
		return "", err
	}
	return ingress.Spec.Domain, nil
}
