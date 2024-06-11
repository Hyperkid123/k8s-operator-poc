/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Hyperkid123/chrome-like/api/v1alpha1"
)

// ChromeDynamicUIReconciler reconciles a ChromeDynamicUI object
type ChromeDynamicUIReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=martin.com,resources=chromedynamicuis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=martin.com,resources=chromedynamicuis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=martin.com,resources=chromedynamicuis/finalizers,verbs=update

// +kubebuilder:rbac:groups=martin.com,resources=chromeuimodules,verbs=get;list;watch
// +kubebuilder:rbac:groups=martin.com,resources=chromeuimodules/status,verbs=get

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=martin.com,resources=chromedynamicuis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=martin.com,resources=chromedynamicuis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=martin.com,resources=chromedynamicuis/finalizers,verbs=update

func (r *ChromeDynamicUIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	// cronJob := &batchv1.CronJob{}
	// // get the cron job

	// err := r.Get(ctx, req.NamespacedName, cronJob)
	// if err != nil {
	// 	log.Error(err, "Failed to fetch CronJob")
	// 	return ctrl.Result{}, client.IgnoreNotFound(err)
	// }

	finalizerName := "finalizer.martin.com"

	dynamicModules := &v1alpha1.ChromeUIModules{}
	dynamicUi := &v1alpha1.ChromeDynamicUI{}
	err := r.Get(ctx, req.NamespacedName, dynamicUi)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("ChromeDynamicUI resource not found")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	log.Info(fmt.Sprintf("Reconciling ChromeDynamicUI; namespace: %s; name: %s", dynamicModules.Namespace, dynamicModules.Name))
	err = r.Get(ctx, types.NamespacedName{
		Namespace: "default",
		Name:      "chrome-service",
	}, dynamicModules)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("ChromeDynamicModule resource not found; requeueing...")
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
		}
		log.Error(err, "Failed to get ChromeDynamicModule resource")
		return ctrl.Result{}, err
	}

	// var configMapVersion string
	// var fedModules []v1alpha1.FedModule
	if dynamicModules.Spec.ConfigMap == "" {
		log.Info("ChromeDynamicModule config map not ready; requeueing...")

		err = fmt.Errorf("ChromeDynamicModule config map name not configured")
		return ctrl.Result{}, err
	}

	uiFedModule := &v1alpha1.FedModule{
		Name:     dynamicUi.Spec.Name,
		UiModule: dynamicUi.Spec.Module,
	}

	configMapName := dynamicModules.Spec.ConfigMap
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: dynamicModules.Namespace}, foundConfigMap)

	if err != nil && !errors.IsNotFound(err) {
		log.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}

	if dynamicUi.ObjectMeta.DeletionTimestamp.IsZero() && errors.IsNotFound(err) {
		// create default config map if it does not exist
		log.Info("ConfigMap not found; Creating default")
		cm, err := createDefaultConfigMap(dynamicModules, uiFedModule, configMapName)
		if err != nil {
			log.Error(err, "Failed to create ConfigMap")
			return ctrl.Result{}, err
		}

		err = r.Create(ctx, cm)
		if err != nil {
			log.Error(err, "Failed to create ConfigMap")
			return ctrl.Result{}, err
		}
		return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
	}

	if err != nil {
		log.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	}
	// if the deletion timestamp is zero, resource is not being delete and we can reconcile as usual
	// if it is not zero, resource has to be removed from the config map
	if dynamicUi.ObjectMeta.DeletionTimestamp.IsZero() {

		if !controllerutil.ContainsFinalizer(dynamicUi, finalizerName) {
			controllerutil.AddFinalizer(dynamicUi, finalizerName)
			err := r.Update(ctx, dynamicUi)
			if err != nil {
				log.Error(err, "Failed to add finalizer")
				return ctrl.Result{}, err
			}
		}
	} else {
		log.Info("Resource scheduled for deletion!")
		if controllerutil.ContainsFinalizer(dynamicUi, finalizerName) {
			log.Info("Deleting external resources")
			err = r.deleteExternalResources(foundConfigMap, uiFedModule)
			if err != nil {
				log.Error(err, "Failed to delete external resources")
				return ctrl.Result{}, err
			}
			err = r.Update(ctx, foundConfigMap)
			if err != nil {
				log.Error(err, "Failed to delete module from ConfigMap")
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(dynamicUi, finalizerName)
			err = r.Update(ctx, dynamicUi)

			if err != nil {
				log.Error(err, "Failed to remove finalizer")
				return ctrl.Result{}, err
			}

		}
		return ctrl.Result{}, nil

	}

	log.Info("Found chrome-service ConfigMap")
	err = updateConfigMap(foundConfigMap, uiFedModule)
	if err != nil {
		log.Error(err, "Failed to update ConfigMap")
		return ctrl.Result{}, err
	}

	err = r.Update(ctx, foundConfigMap)
	if err != nil {
		log.Error(err, "Failed to update ConfigMap")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func serializeConfigMapData(fedModules []v1alpha1.FedModule) (string, error) {
	b, err := json.Marshal(fedModules)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func createDefaultConfigMap(uiModules *v1alpha1.ChromeUIModules, baseModule *v1alpha1.FedModule, configMapName string) (*corev1.ConfigMap, error) {
	cm := &corev1.ConfigMap{}
	defaultFedModules := []v1alpha1.FedModule{}
	for _, def := range *uiModules.Spec.UIModuleTemplates {
		if def == baseModule.Name {
			defaultFedModules = append(defaultFedModules, *baseModule)
			break
		}
	}

	b, err := serializeConfigMapData(defaultFedModules)
	if err != nil {
		return nil, err
	}
	s := string(b)
	cm.APIVersion = "v1"
	cm.Namespace = uiModules.Namespace
	cm.Kind = "ConfigMap"
	cm.Name = configMapName
	cm.Data = map[string]string{
		"fed-modules": s,
	}

	return cm, nil
}

func updateConfigMap(cm *corev1.ConfigMap, affectedModule *v1alpha1.FedModule) error {
	currentFedModules := []v1alpha1.FedModule{}
	err := json.Unmarshal([]byte(cm.Data["fed-modules"]), &currentFedModules)
	if err != nil {
		return err
	}

	found := false
	for i, module := range currentFedModules {
		if module.Name == affectedModule.Name {
			currentFedModules[i] = *affectedModule
			found = true
			break
		}
	}

	if !found {
		currentFedModules = append(currentFedModules, *affectedModule)
	}

	b, err := serializeConfigMapData(currentFedModules)
	if err != nil {
		return err
	}

	cm.Data["fed-modules"] = string(b)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChromeDynamicUIReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ChromeDynamicUI{}).
		Complete(r)
}

func (r *ChromeDynamicUIReconciler) deleteExternalResources(configMap *corev1.ConfigMap, dynamicUi *v1alpha1.FedModule) error {
	removeIndex := -1
	currentModules := []v1alpha1.FedModule{}
	err := json.Unmarshal([]byte(configMap.Data["fed-modules"]), &currentModules)
	if err != nil {
		return err
	}

	for index, module := range currentModules {
		if module.Name == dynamicUi.Name {
			removeIndex = index
			break
		}
	}

	if removeIndex >= 0 {
		currentModules = append(currentModules[:removeIndex], currentModules[removeIndex+1:]...)
	}

	b, err := serializeConfigMapData(currentModules)
	if err != nil {
		return err
	}

	configMap.Data["fed-modules"] = string(b)

	fmt.Println("Deleting external resources", dynamicUi)
	return nil
}
