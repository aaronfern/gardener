// Copyright 2023 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gardenerscheduler

import (
	coordinationv1beta1 "k8s.io/api/coordination/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	schedulerv1alpha1 "github.com/gardener/gardener/pkg/scheduler/apis/config/v1alpha1"
)

const (
	clusterRoleName        = "gardener.cloud:system:scheduler"
	clusterRoleBindingName = "gardener.cloud:system:scheduler"
)

func (g *gardenerScheduler) clusterRole() *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:   clusterRoleName,
			Labels: GetLabels(),
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"create", "patch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"create", "delete", "get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{gardencorev1beta1.GroupName},
				Resources: []string{
					"cloudprofiles",
					"seeds",
				},
				Verbs: []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{gardencorev1beta1.GroupName},
				Resources: []string{
					"shoots",
				},
				Verbs: []string{"get", "list", "watch", "patch", "update"},
			},
			{
				APIGroups: []string{gardencorev1beta1.GroupName},
				Resources: []string{
					"shoots/binding",
				},
				Verbs: []string{"update"},
			},
			{
				APIGroups: []string{coordinationv1beta1.GroupName},
				Resources: []string{
					"leases",
				},
				Verbs: []string{"create"},
			},
			{
				APIGroups: []string{coordinationv1beta1.GroupName},
				Resources: []string{
					"leases",
				},
				ResourceNames: []string{
					schedulerv1alpha1.SchedulerDefaultLockObjectName,
				},
				Verbs: []string{"get", "watch", "update"},
			},
		},
	}
}

func (g *gardenerScheduler) clusterRoleBinding(serviceAccountName string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:   clusterRoleBindingName,
			Labels: GetLabels(),
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     clusterRoleName,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      serviceAccountName,
			Namespace: metav1.NamespaceSystem,
		}},
	}
}
