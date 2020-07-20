package controllers

import (
	"context"
	"fmt"
	infrav1 "github.com/redwarn/kubeapp/api/v1"
	"reflect"
	"strings"

	"github.com/containous/traefik/v2/pkg/provider/kubernetes/crd/traefik/v1alpha1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AppReconciler) reconclieIngressRoute(app *infrav1.App) error {
	log := r.Log

	if app.Spec.Domain == "" {
		return nil
	}

	ingRoute := r.genIngressRoute(app)

	if err := controllerutil.SetControllerReference(app, ingRoute, r.Scheme); err != nil {
		log.Error(err, "set App ControllerReference Error")
		return err
	}

	found := &v1alpha1.IngressRoute{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, found)

	if err != nil && apierrs.IsNotFound(err) {
		log.Info("ingreeRoute NotFound and Creating new one")
		if err = r.Create(context.TODO(), ingRoute); err != nil {
			return err
		}

	} else if err != nil {

		log.Error(err, "Get svc info Error")
		return err

	} else if !reflect.DeepEqual(ingRoute.Spec, found.Spec) {
		// Update the found object and write the result back if there are any changes
		found.Spec = ingRoute.Spec
		log.Info("old ingreeRoute changed and Updating ingreeRoute to reconcile", "name", ingRoute.Name)
		err = r.Update(context.TODO(), found)
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	ingressRouteApiVersion = "traefik.containo.us/v1alpha1"
	ingressRouteKind       = "IngressRoute"
)

func (r *AppReconciler) genIngressRoute(app *infrav1.App) *v1alpha1.IngressRoute {
	var port int32
	labels := map[string]string{
		"app":  app.Spec.Name,
		"unit": app.Spec.Unit,
	}
	for _, p := range app.Spec.Ports {
		if p.Name == bizPortName {
			port = p.ContainerPort
		}
	}
	matchName := genRouteMatchName(app)
	ingressRoute := &v1alpha1.IngressRoute{
		TypeMeta: metav1.TypeMeta{Kind: ingressRouteKind, APIVersion: ingressRouteApiVersion},
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: v1alpha1.IngressRouteSpec{
			Routes: []v1alpha1.Route{
				{
					Match:    matchName,
					Kind:     "Rule",
					Priority: 0,
					Services: []v1alpha1.Service{
						{
							LoadBalancerSpec: v1alpha1.LoadBalancerSpec{
								Namespace: app.Namespace,
								Name:      app.Name,
								Port:      port,
							},
						},
					},
					Middlewares: app.Spec.Middlewares,
				},
			},
		},
	}
	return ingressRoute
}

func genRouteMatchName(app *infrav1.App) string {
	hosts := strings.Split(app.Spec.Domain, ",")
	var domain string
	for i, host := range hosts {
		if i == 0 {
			domain = fmt.Sprintf("Host(`%s`)", host)
		} else {
			domain = fmt.Sprintf("%s || Host(`%s`)", domain, host)
		}
	}
	if app.Spec.Unit == "blue" {
		return fmt.Sprintf("(%s) && PathPrefix(`%s`)", domain, app.Spec.Path)
	}
	return fmt.Sprintf("(%s) && PathPrefix(`%s`) && HeadersRegexp(`oriente-agent`,`%s`)", domain, app.Spec.Path, app.Spec.Unit)
}
