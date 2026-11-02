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

package bcs

import (
	"context"
	"errors"

	"github.com/bytedance/mockey"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbcs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

var _ = Describe("ProjectManager", func() {
	var (
		ctx     context.Context
		user    auth.User
		manager *ProjectManager
	)

	BeforeEach(func() {
		ctx = context.Background()
		user = auth.User{ID: "tester"}
		manager = NewProjectManager()
	})

	It("loads a bindable existing project", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).Return(&cloudbcs.Project{
				ID:    "bcs-uid",
				Code:  "existing-bcs",
				Kind:  cloudbcs.ProjectKindK8s,
				BizID: "398",
			}, nil).Build()

			project, err := manager.LoadBoundProject(ctx, "existing-bcs")
			Expect(err).NotTo(HaveOccurred())
			Expect(project.Code).To(Equal("existing-bcs"))
			Expect(project.BizID).To(Equal("398"))
		})
	})

	It("rejects binding a non-k8s project", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).Return(&cloudbcs.Project{
				ID:   "bcs-uid",
				Code: "existing-bcs",
				Kind: "mesos",
			}, nil).Build()

			_, err := manager.LoadBoundProject(ctx, "existing-bcs")
			Expect(err).To(MatchError(ContainSubstring("is not a k8s project")))
		})
	})

	It("reuses the existing deterministic project when it already exists", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).Return(&cloudbcs.Project{
				ID:    "bcs-uid",
				Code:  "bkms-ws",
				Kind:  cloudbcs.ProjectKindK8s,
				BizID: "398",
			}, nil).Build()
			mockey.Mock((*cloudbcs.StubApiClient).CreateProject).
				Return(nil, errors.New("should not create when project already exists")).
				Build()

			project, err := manager.EnsureProject(ctx, "workspace-name", "bkms-ws", 398)
			Expect(err).NotTo(HaveOccurred())
			Expect(project.ID).To(Equal("bcs-uid"))
			Expect(project.Code).To(Equal("bkms-ws"))
		})
	})

	It("creates the deterministic project when it does not exist", func() {
		mockey.PatchConvey("test", GinkgoT(), func() {
			mockey.Mock(auth.MustGetUser).Return(user).Build()
			mockey.Mock(cloudbcs.New).Return(cloudbcs.NewStub(user), nil).Build()
			mockey.Mock((*cloudbcs.StubApiClient).GetProject).Return(nil, cloudbcs.ErrProjectNotFound).Build()
			mockey.Mock((*cloudbcs.StubApiClient).CreateProject).Return(&cloudbcs.Project{
				ID:    "new-bcs-uid",
				Code:  "bkms-ws",
				Name:  "workspace-name",
				Kind:  cloudbcs.ProjectKindK8s,
				BizID: "398",
			}, nil).Build()

			project, err := manager.EnsureProject(ctx, "workspace-name", "bkms-ws", 398)
			Expect(err).NotTo(HaveOccurred())
			Expect(project.ID).To(Equal("new-bcs-uid"))
			Expect(project.BizID).To(Equal("398"))
		})
	})
})
