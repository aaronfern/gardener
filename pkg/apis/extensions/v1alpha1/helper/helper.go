// Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package helper

import (
	"net"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	ca "github.com/gardener/gardener/pkg/component/clusterautoscaler"
)

// ClusterAutoscalerRequired returns whether the given worker pool configuration indicates that a cluster-autoscaler
// is needed.
func ClusterAutoscalerRequired(pools []extensionsv1alpha1.WorkerPool) bool {
	for _, pool := range pools {
		if pool.Maximum > pool.Minimum {
			return true
		}
	}
	return false
}

// GetDNSRecordType returns the appropriate DNS record type (A/AAAA or CNAME) for the given address.
func GetDNSRecordType(address string) extensionsv1alpha1.DNSRecordType {
	if ip := net.ParseIP(address); ip != nil {
		if ip.To4() != nil {
			return extensionsv1alpha1.DNSRecordTypeA
		}
		return extensionsv1alpha1.DNSRecordTypeAAAA
	}
	return extensionsv1alpha1.DNSRecordTypeCNAME
}

// GetDNSRecordTTL returns the value of the given ttl, or 120 if nil.
func GetDNSRecordTTL(ttl *int64) int64 {
	if ttl != nil {
		return *ttl
	}
	return 120
}

// DeterminePrimaryIPFamily determines the primary IP family out of a specified list of IP families.
func DeterminePrimaryIPFamily(ipFamilies []extensionsv1alpha1.IPFamily) extensionsv1alpha1.IPFamily {
	if len(ipFamilies) == 0 {
		return extensionsv1alpha1.IPFamilyIPv4
	}
	return ipFamilies[0]
}

// FilePathsFrom returns the paths for all the given files.
func FilePathsFrom(files []extensionsv1alpha1.File) []string {
	var out []string

	for _, file := range files {
		out = append(out, file.Path)
	}

	return out
}

// GetClusterAutoscalerAnnotationMap returns a map of annotations with values intended to be used as cluster autoscaler options for the worker group
func GetClusterAutoscalerAnnotationMap(caOptions *extensionsv1alpha1.ClusterAutoscalerOptions) map[string]string {
	mcdAnnotationMap := map[string]string{}
	if caOptions != nil {
		if caOptions.ScaleDownUtilizationThreshold != nil {
			mcdAnnotationMap[ca.ScaleDownUtilizationThresholdAnnotation] = *caOptions.ScaleDownUtilizationThreshold
		}
		if caOptions.ScaleDownGpuUtilizationThreshold != nil {
			mcdAnnotationMap[ca.ScaleDownGpuUtilizationThresholdAnnotation] = *caOptions.ScaleDownGpuUtilizationThreshold
		}
		if caOptions.ScaleDownUnneededTime != nil {
			mcdAnnotationMap[ca.ScaleDownUnneededTimeAnnotation] = caOptions.ScaleDownUnneededTime.Duration.String()
		}
		if caOptions.ScaleDownUnreadyTime != nil {
			mcdAnnotationMap[ca.ScaleDownUnreadyTimeAnnotation] = caOptions.ScaleDownUnreadyTime.Duration.String()
		}
		if caOptions.MaxNodeProvisionTime != nil {
			mcdAnnotationMap[ca.MaxNodeProvisionTimeAnnotation] = caOptions.MaxNodeProvisionTime.Duration.String()
		}
	}

	return mcdAnnotationMap
}
