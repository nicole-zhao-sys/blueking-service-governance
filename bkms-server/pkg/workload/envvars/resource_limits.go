/*
 * TencentBlueKing is pleased to support the open source community by making
 * 蓝鲸智云 - 服务治理 (BlueKing Service Governance) available.
 * Copyright (C) Tencent. All rights reserved.
 * Licensed under the MIT License (the "License"); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 *
 *  http://opensource.org/licenses/MIT
 *
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 *
 * We undertake not to change the open source license (MIT license) applicable
 * to the current version of the project delivered to anyone in the future.
 */

package envvars

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

// resourceLimitBuiltinVars builds builtin env vars from a workload resource map.
// The map value is "{requests}" or "{requests}-{limits}". The env value is the
// original limit string after ParseQuantity succeeds, so units such as "0.2"
// and "2048Mi" are preserved instead of the canonical Quantity form.
func resourceLimitBuiltinVars(resources map[string]string) envvartypes.EnvVariableList {
	if len(resources) == 0 {
		return nil
	}

	specs := []struct {
		resourceName string
		envName      string
		description  string
	}{
		{string(corev1.ResourceCPU), EnvVarNameCPULimit, "The CPU limit configured for the workload"},
		{string(corev1.ResourceMemory), EnvVarNameMemoryLimit, "The memory limit configured for the workload"},
	}

	vars := make(envvartypes.EnvVariableList, 0, len(specs))
	for _, spec := range specs {
		raw, ok := resources[spec.resourceName]
		if !ok {
			continue
		}
		limit, ok := limitQuantityString(raw)
		if !ok {
			continue
		}
		vars = append(vars, envvartypes.EnvVariableObj{
			Key:         spec.envName,
			Value:       limit,
			Description: spec.description,
			IsBuiltin:   true,
		})
	}
	if len(vars) == 0 {
		return nil
	}
	return vars
}

// limitQuantityString returns the limit segment of a workload resource value.
// A value without "-" is both the request and the limit.
func limitQuantityString(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", false
	}

	_, after, found := strings.Cut(raw, "-")
	limit := raw
	if found {
		limit = strings.TrimSpace(after)
	}
	if limit == "" {
		return "", false
	}
	if _, err := resource.ParseQuantity(limit); err != nil {
		return "", false
	}
	return limit, true
}
