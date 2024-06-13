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
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1alpha1 "github.com/Hyperkid123/chrome-like/api/v1alpha1"
)

// ChromeUIModulesReconciler reconciles a ChromeUIModules object
type ChromeUIModulesReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=martin.com,resources=chromeuimodules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=martin.com,resources=chromeuimodules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=martin.com,resources=chromeuimodules/finalizers,verbs=update

// +kubebuilder:rbac:groups="martin.com",resources=configmaps,verbs=get;list;watch;create;update;patch

// +kubebuilder:rbac:groups="martin.com",resources=deployments,verbs=list;watch;update;patch

// +kubebuilder:rbac:groups=martin.com,resources=chromedynamicuis,verbs=get;list;watch;create;update;patch;delete
func (r *ChromeUIModulesReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)
	dynamicModules := &v1alpha1.ChromeUIModules{}
	uiModules := &v1alpha1.ChromeDynamicUIList{}

	if err := r.Get(ctx, req.NamespacedName, dynamicModules); err != nil {
		log.Error(err, "unable to get resource ChromeUIModules")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	err := r.List(ctx, uiModules)
	if err != nil {
		log.Error(err, "unable to list ChromeDynamicUI")
		return ctrl.Result{}, err
	}

	labels := map[string]string{
		"chrome-operator-reload": "true",
	}

	listOpts := []client.ListOption{
		client.InNamespace("default"),
		client.MatchingLabels(labels),
	}

	deploymentList := &appsv1.DeploymentList{}

	err = r.List(ctx, deploymentList, listOpts...)

	if err != nil {
		log.Error(err, "unable to list Deployments")
		return ctrl.Result{}, err
	}

	configMapName := dynamicModules.Spec.ConfigMap
	foundConfigMap := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: configMapName, Namespace: dynamicModules.Namespace}, foundConfigMap)

	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("ConfigMap chrome-service not found. Creating a new one")
			cm := createEmptyConfigMap(configMapName)
			r.Create(ctx, cm)
			return ctrl.Result{}, nil
		}

		log.Error(err, "Error fetching ConfigMap")
		return ctrl.Result{}, err
	}

	newFedModules, err := validateConfigMap(uiModules, *dynamicModules.Spec.UIModuleTemplates)
	if err != nil {
		log.Error(err, "Error validating ConfigMap")
		return ctrl.Result{}, err
	}

	b, err := json.Marshal(newFedModules)
	if err != nil {
		log.Error(err, "Error marshalling ConfigMap")
		return ctrl.Result{}, err
	}

	foundConfigMap.Data["fed-modules"] = string(b)

	err = r.Update(ctx, foundConfigMap)
	if err != nil {
		log.Error(err, "Error updating ConfigMap")
		return ctrl.Result{}, err
	}

	log.Info("ConfigMap updated; Updating ui modules")
	log.Info(fmt.Sprintln(*dynamicModules.Spec.UIModuleTemplates))
	log.Info(fmt.Sprintln("Number of UI modules: ", len(uiModules.Items)))

	for _, deployment := range deploymentList.Items {
		// log.Info(fmt.Sprintln("*******************, Deployment: ", deployment.Labels, "Name: ", deployment.Name, "Should reload: ", deployment.Labels["chrome-operator-reload"], "Anotations: ", deployment.Annotations))
		if deployment.Labels["chrome-operator-reload"] == "true" {
			checksum, err := generateConfigMapChecksum(foundConfigMap.Data["fed-modules"])
			if err != nil {
				log.Error(err, "Error generating checksum")
				return ctrl.Result{}, err
			}
			deployment.Annotations["chrome-operator-checksum/configmap"] = checksum
			if deployment.Spec.Template.ObjectMeta.Annotations == nil {
				deployment.Spec.Template.ObjectMeta.Annotations = map[string]string{}
			}
			deployment.Spec.Template.ObjectMeta.Annotations["chrome-operator-checksum/configmap"] = checksum
			err = r.Update(ctx, &deployment)
			if err != nil {
				log.Error(err, "Error updating Deployment")
				return ctrl.Result{}, err
			}

			log.Info(fmt.Sprintln("Updated Deployment: ", deployment.Name, " with checksum: ", checksum))
		}
	}

	return ctrl.Result{}, nil
}

type validTemplates struct {
	Name  string
	Index int
}

func generateConfigMapChecksum(value string) (string, error) {
	hasher := sha1.New()
	_, err := hasher.Write([]byte(value))
	if err != nil {
		return "", err
	}

	checksum := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	return checksum, nil
}

func validateConfigMap(uiModules *v1alpha1.ChromeDynamicUIList, availableTemplates []string) ([]v1alpha1.FedModule, error) {
	indexMap := []validTemplates{}
	for _, template := range availableTemplates {
		for index, module := range uiModules.Items {
			if module.Name == template {
				indexMap = append(indexMap, validTemplates{Name: module.Name, Index: index})
				break
			}
		}
	}

	newFedModules := []v1alpha1.FedModule{}
	for _, value := range indexMap {
		newModule := v1alpha1.FedModule{
			UiModule: uiModules.Items[value.Index].Spec.Module,
			Name:     uiModules.Items[value.Index].Spec.Name,
		}
		newFedModules = append(newFedModules, newModule)
	}

	return newFedModules, nil
}

func createEmptyConfigMap(configMapName string) *corev1.ConfigMap {
	cm := &corev1.ConfigMap{}
	cm.APIVersion = "v1"
	cm.Namespace = "default"
	cm.Kind = "ConfigMap"
	cm.Name = configMapName
	cm.Data = map[string]string{
		"fed-modules": "[]",
	}
	return &corev1.ConfigMap{}

}

// SetupWithManager sets up the controller with the Manager.
func (r *ChromeUIModulesReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.ChromeUIModules{}).
		Complete(r)
}
