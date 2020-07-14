package controllers

import (
	"context"
	infrav1 "kubeapp/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

func (r *AppReconciler) syncStatus(app *infrav1.App) error {
	ctx := context.Background()
	log := r.Log.WithValues("app-status", app.Namespace)
	if app.Status.Replicas != 0 && app.Status.Replicas == app.Status.UpdatedReplicas && app.Status.Replicas == app.Status.AvailableReplicas {
		return nil
	}

	deploy := &appsv1.Deployment{}
	err := r.Get(ctx, types.NamespacedName{Name: app.Name, Namespace: app.Namespace}, deploy)

	if err != nil && apierrs.IsNotFound(err) {
		return nil
	} else if err != nil {
		log.Error(err, "get deployment info Error", "name", app.Name)
		return err
	}

	status := infrav1.AppStatus{
		Replicas:            app.Spec.Replicas,
		ReadyReplicas:       deploy.Status.AvailableReplicas,
		UpdatedReplicas:     deploy.Status.UpdatedReplicas,
		AvailableReplicas:   deploy.Status.AvailableReplicas,
		UnavailableReplicas: deploy.Status.UnavailableReplicas,
	}
	app.Status = status
	if err := r.Update(ctx, app); err != nil {
		log.Error(err, "update app status error", "name", app.Name)
	}
	return nil
}
