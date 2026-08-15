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

package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/gin-gonic/gin"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	modelbkci "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/bkci"
	slz "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/bkintegrations/serializer"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/bkerrs"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/common/config"
	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbkci "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bkci"
	storereg "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/server/registry"
)

type mockBkCIProjectStore struct {
	getByWorkspaceFunc func(ctx context.Context, workspaceID string) (*modelbkci.Project, error)
}

func (m *mockBkCIProjectStore) Create(_ context.Context, _ *modelbkci.Project) error { return nil }

func (m *mockBkCIProjectStore) GetByWorkspace(ctx context.Context, workspaceID string) (*modelbkci.Project, error) {
	if m.getByWorkspaceFunc != nil {
		return m.getByWorkspaceFunc(ctx, workspaceID)
	}
	return nil, modelbkci.ErrProjectNotFound
}

var _ = Describe("BkCI repository ref adapters", func() {
	Describe("newBkCIRepositoryRefsOutput", func() {
		It("should convert repository refs into serializer outputs", func() {
			outputs := newBkCIRepositoryRefsOutput([]cloudbkci.RepositoryRef{
				{
					Name:    "main",
					Path:    "refs/heads/main",
					SHA:     "abc123",
					LinkURL: "https://example.com/branch/main",
				},
				{
					Name:    "v1.0.0",
					Path:    "refs/tags/v1.0.0",
					SHA:     "def456",
					LinkURL: "https://example.com/tag/v1.0.0",
				},
			})

			Expect(outputs).To(Equal([]*slz.BkCIRepositoryRefOutput{
				{
					Name:    "main",
					Path:    "refs/heads/main",
					SHA:     "abc123",
					LinkURL: "https://example.com/branch/main",
				},
				{
					Name:    "v1.0.0",
					Path:    "refs/tags/v1.0.0",
					SHA:     "def456",
					LinkURL: "https://example.com/tag/v1.0.0",
				},
			}))

			payload, err := json.Marshal(outputs)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(payload)).To(ContainSubstring(`"name":"main"`))
			Expect(string(payload)).To(ContainSubstring(`"path":"refs/heads/main"`))
			Expect(string(payload)).To(ContainSubstring(`"linkUrl":"https://example.com/tag/v1.0.0"`))
		})
	})
})

var _ = Describe("BkCI repository ref handlers", func() {
	var (
		handler   *Handler
		router    *gin.Engine
		oldConfig *config.Config
	)

	BeforeEach(func() {
		oldConfig = config.G
		config.G = &config.Config{
			Development: config.DevConfig{
				UseStubBkCI: true,
			},
		}

		handler = New(&storereg.Registry{
			BkCIProjectStore: &mockBkCIProjectStore{
				getByWorkspaceFunc: func(_ context.Context, workspaceID string) (*modelbkci.Project, error) {
					return &modelbkci.Project{
						ID:          "project-1",
						Code:        "demo-project",
						WorkspaceID: workspaceID,
					}, nil
				},
			},
		})

		gin.SetMode(gin.TestMode)
		router = gin.New()
		router.Use(func(c *gin.Context) {
			ctx := auth.WithUser(c.Request.Context(), auth.User{
				ID: "tester",
				Cred: auth.UserCredential{
					AccessToken: "token",
				},
			})
			c.Request = c.Request.WithContext(ctx)
			c.Next()
		})
		router.Use(bkerrs.ErrorHandler())
	})

	AfterEach(func() {
		config.G = oldConfig
	})

	Describe("ListBkCIRepositoryBranches", func() {
		It("should return repository branches", func() {
			router.GET("/workspaces/:workspaceID/bkci-repositories/branches", handler.ListBkCIRepositoryBranches)

			req := httptest.NewRequest(
				http.MethodGet,
				"/workspaces/ws-1/bkci-repositories/branches?repositoryID=repo-1&repositoryType=ID&search=release&page=1&pageSize=20",
				nil,
			)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))

			var resp slz.ListBkCIRepositoryBranchesOutput
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).To(Equal([]*slz.BkCIRepositoryRefOutput{
				{
					Name:    "release",
					Path:    "refs/heads/release",
					SHA:     "stub-branch-sha",
					LinkURL: "https://git.example.com/stub-group/stub-repo/tree/release",
				},
			}))
		})
	})

	Describe("ListBkCIRepositoryTags", func() {
		It("should return repository tags", func() {
			router.GET("/workspaces/:workspaceID/bkci-repositories/tags", handler.ListBkCIRepositoryTags)

			req := httptest.NewRequest(
				http.MethodGet,
				"/workspaces/ws-1/bkci-repositories/tags?repositoryID=repo-1&repositoryType=NAME&search=v1.2.3&page=1&pageSize=20",
				nil,
			)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			Expect(rec.Code).To(Equal(http.StatusOK))

			var resp slz.ListBkCIRepositoryTagsOutput
			Expect(json.Unmarshal(rec.Body.Bytes(), &resp)).To(Succeed())
			Expect(resp.Data).To(Equal([]*slz.BkCIRepositoryRefOutput{
				{
					Name:    "v1.2.3",
					Path:    "refs/tags/v1.2.3",
					SHA:     "stub-tag-sha",
					LinkURL: "https://git.example.com/stub-group/stub-repo/releases/tag/v1.2.3",
				},
			}))
		})
	})
})
