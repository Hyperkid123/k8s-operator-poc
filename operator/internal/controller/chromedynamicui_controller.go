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
func (r *ChromeDynamicUIReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	dynamicModules := &v1alpha1.ChromeUIModules{}
	dynamicUi := &v1alpha1.ChromeDynamicUI{}

	// reference to different CRDs
	// if err := ctrl.SetControllerReference(dynamicUi, dynamicModules, r.Scheme); err != nil {
	// 	return ctrl.Result{}, err
	// }

	err := r.Get(ctx, req.NamespacedName, dynamicUi)
	if err != nil {
		log.Error(err, "Failed to get ChromeDynamicUI resource")
		return ctrl.Result{}, err
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
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("ConfigMap not found; Creating default")
			defaultFedModules := []v1alpha1.FedModule{}
			for _, def := range *dynamicModules.Spec.UIModuleTemplates {
				if def == uiFedModule.Name {
					defaultFedModules = append(defaultFedModules, *uiFedModule)
					break
				}
			}

			b, err := json.Marshal(defaultFedModules)
			if err != nil {
				log.Error(err, "Failed to marshal defaultFedModules")
				return ctrl.Result{}, err
			}
			s := string(b)
			foundConfigMap.APIVersion = "v1"
			foundConfigMap.Namespace = dynamicModules.Namespace
			foundConfigMap.Kind = "ConfigMap"
			foundConfigMap.Name = configMapName
			foundConfigMap.Data = map[string]string{
				"fed-modules": s,
			}

			err = r.Create(ctx, foundConfigMap)
			if err != nil {
				log.Error(err, "Failed to create ConfigMap")
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true, RequeueAfter: time.Second * 5}, nil
		}
		log.Error(err, "Failed to get ConfigMap")
		return ctrl.Result{}, err
	} else {
		log.Info("Found chrome-service ConfigMap")
		currentFedModules := []v1alpha1.FedModule{}
		err = json.Unmarshal([]byte(foundConfigMap.Data["fed-modules"]), &currentFedModules)
		if err != nil {
			log.Error(err, "Failed to unmarshal fed-modules")
			return ctrl.Result{}, err
		}

		found := false
		for i, module := range currentFedModules {
			if module.Name == uiFedModule.Name {
				currentFedModules[i] = *uiFedModule
				found = true
				break
			}
		}

		if !found {
			currentFedModules = append(currentFedModules, *uiFedModule)
		}

		b, err := json.Marshal(currentFedModules)
		if err != nil {
			log.Error(err, "Failed to marshal currentFedModules")
			return ctrl.Result{}, err
		}

		foundConfigMap.Data["fed-modules"] = string(b)

		err = r.Update(ctx, foundConfigMap)
		if err != nil {
			log.Error(err, "Failed to update ConfigMap")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ChromeDynamicUIReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ChromeDynamicUI{}).
		Complete(r)
}
