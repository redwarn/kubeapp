package controllers

import (
	"context"
	infrav1 "github.com/redwarn/kubeapp/api/v1"
	"reflect"

	v1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AppReconciler) reconcileService(app *infrav1.App) error {
	log := r.Log
	svc := r.genService(app)

	if err := controllerutil.SetControllerReference(app, svc, r.Scheme); err != nil {
		log.Error(err, "Set App ControllerReference Error")
		return err
	}

	found := &v1.Service{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, found)

	if err != nil && apierrs.IsNotFound(err) {
		log.Info("Service NotFound and Creating new one")
		if err = r.Create(context.TODO(), svc); err != nil {
			return err
		}

	} else if err != nil {

		log.Error(err, "Get svc info Error")
		return err

	} else if !reflect.DeepEqual(svc.Spec, found.Spec) {
		clusterIP := found.Spec.ClusterIP
		// Update the found object and write the result back if there are any changes
		found.Spec = svc.Spec
		found.Spec.ClusterIP = clusterIP
		log.Info("old svc changed and Updating svc to reconcile", "name", svc.Name)
		err = r.Update(context.TODO(), found)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r *AppReconciler) genService(app *infrav1.App) *v1.Service {
	labels := map[string]string{
		"app":  app.Spec.Name,
		"unit": app.Spec.Unit,
	}
	var ports []v1.ServicePort
	for _, p := range app.Spec.Ports {
		if p.Name == "web" {
			ports = append(ports, v1.ServicePort{
				Name:       p.Name,
				Port:       p.ServicePort,
				Protocol:   v1.Protocol(p.Protocol),
				TargetPort: intstr.FromInt(int(p.ContainerPort)),
			})
		}

	}

	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      app.Name,
			Namespace: app.Namespace,
			Labels:    labels,
		},
		Spec: v1.ServiceSpec{
			Selector:        labels,
			Ports:           ports,
			ClusterIP:       "",
			Type:            v1.ServiceTypeClusterIP,
			SessionAffinity: v1.ServiceAffinityNone,
		},
	}
}
