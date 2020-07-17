package controllers

import (
	"context"
	infrav1 "kubeapp/api/v1"
	"reflect"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"

	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func (r *AppReconciler) reconclieMonitor(app *infrav1.App) error {
	log := r.Log

	if app.Spec.Metrics == "" {
		return nil
	}

	podMinitor := genPodMonitor(app)

	if err := controllerutil.SetControllerReference(app, podMinitor, r.Scheme); err != nil {
		log.Error(err, "set App ControllerReference Error")
		return err
	}

	found := &monitoringv1.PodMonitor{}
	err := r.Get(context.TODO(), types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, found)

	if err != nil && apierrs.IsNotFound(err) {
		log.Info("podMonitor NotFound and Creating new one")
		if err = r.Create(context.TODO(), podMinitor); err != nil {
			return err
		}

	} else if err != nil {
		log.Error(err, "get podMonitor info Error")
		return err

	} else if !reflect.DeepEqual(podMinitor.Spec, found.Spec) {
		// Update the found object and write the result back if there are any changes
		found.Spec = podMinitor.Spec
		log.Info("old podMonitor changed and Updating podMonitor to reconcile", "name", app.Name)
		err = r.Update(context.TODO(), found)
		if err != nil {
			return err
		}
	}
	return nil
}

const (
	podMonitorApiVersion = "onitoring.coreos.com/v1"
	podMonitorKind       = "PodMonitor"
	metricsPortName      = "metrics"
)

func genPodMonitor(app *infrav1.App) *monitoringv1.PodMonitor {
	port := getPort(app, metricsPortName)
	return &monitoringv1.PodMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: podMonitorApiVersion,
			Kind:       podMonitorKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: app.Namespace,
			Name:      app.Name,
			Labels: map[string]string{
				"team": "biz-app",
			},
		},
		Spec: monitoringv1.PodMonitorSpec{
			Selector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app":  app.Spec.Name,
					"unit": app.Spec.Unit,
				},
			},
			PodMetricsEndpoints: []monitoringv1.PodMetricsEndpoint{
				{
					Path:       app.Spec.Metrics,
					TargetPort: &port,
				},
			},
		},
	}
}
