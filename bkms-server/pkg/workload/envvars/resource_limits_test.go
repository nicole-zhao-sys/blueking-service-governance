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
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/resource"

	envvartypes "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/workload/envvars/types"
)

func TestResourceLimitBuiltinVars(t *testing.T) {
	t.Run("keeps the configured limit unit", func(t *testing.T) {
		g := NewWithT(t)
		vars := resourceLimitBuiltinVars(map[string]string{
			"cpu":    "100m-200m",
			"memory": "512Mi-2Gi",
		})

		g.Expect(vars).To(Equal(envvartypes.EnvVariableList{
			{
				Key:         EnvVarNameCPULimit,
				Value:       "200m",
				Description: "The CPU limit configured for the workload",
				IsBuiltin:   true,
			},
			{
				Key:         EnvVarNameMemoryLimit,
				Value:       "2Gi",
				Description: "The memory limit configured for the workload",
				IsBuiltin:   true,
			},
		}))
	})

	t.Run("preserves a single quantity and does not canonicalize the unit", func(t *testing.T) {
		g := NewWithT(t)
		vars := resourceLimitBuiltinVars(map[string]string{
			"cpu":    "0.2",
			"memory": "2048Mi",
		})

		cpuQuantity := resource.MustParse("0.2")
		memoryQuantity := resource.MustParse("2048Mi")
		g.Expect(cpuQuantity.String()).NotTo(Equal("0.2"))
		g.Expect(memoryQuantity.String()).NotTo(Equal("2048Mi"))
		g.Expect(limitVarByKey(vars, EnvVarNameCPULimit).Value).To(Equal("0.2"))
		g.Expect(limitVarByKey(vars, EnvVarNameMemoryLimit).Value).To(Equal("2048Mi"))
		g.Expect(limitVarByKey(vars, EnvVarNameCPULimit).ValueFrom).To(BeNil())
		g.Expect(limitVarByKey(vars, EnvVarNameCPULimit).Placeholder).To(BeEmpty())
	})

	t.Run("injects only the resource that is configured", func(t *testing.T) {
		g := NewWithT(t)
		vars := resourceLimitBuiltinVars(map[string]string{"memory": "256Mi"})

		g.Expect(vars).To(HaveLen(1))
		g.Expect(vars[0].Key).To(Equal(EnvVarNameMemoryLimit))
		g.Expect(vars[0].Value).To(Equal("256Mi"))
	})

	t.Run("returns nothing when resources are unset or the limit is not a quantity", func(t *testing.T) {
		g := NewWithT(t)
		g.Expect(resourceLimitBuiltinVars(nil)).To(BeNil())
		g.Expect(resourceLimitBuiltinVars(map[string]string{})).To(BeNil())
		g.Expect(resourceLimitBuiltinVars(map[string]string{
			"cpu":    "100m-bad",
			"memory": "2Gi",
		})).To(Equal(envvartypes.EnvVariableList{{
			Key:         EnvVarNameMemoryLimit,
			Value:       "2Gi",
			Description: "The memory limit configured for the workload",
			IsBuiltin:   true,
		}}))
		g.Expect(resourceLimitBuiltinVars(map[string]string{"cpu": "100m-"})).To(BeNil())
	})
}

func limitVarByKey(vars envvartypes.EnvVariableList, key string) envvartypes.EnvVariableObj {
	for _, item := range vars {
		if item.Key == key {
			return item
		}
	}
	return envvartypes.EnvVariableObj{}
}
