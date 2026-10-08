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

	"github.com/pkg/errors"
	"github.com/spf13/cast"

	"github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/account/auth"
	cloudbcs "github.com/TencentBlueKing/blueking-service-governance/bkms-server/pkg/infras/cloudapi/bcs"
)

// ProjectManager 负责 BCS 项目的查询、校验与幂等 ensure。
type ProjectManager struct{}

func NewProjectManager() *ProjectManager {
	return &ProjectManager{}
}

// LoadBoundProject 加载一个已存在且可绑定的 BCS 项目。
func (m *ProjectManager) LoadBoundProject(ctx context.Context, projectCode string) (*cloudbcs.Project, error) {
	client, err := m.newClient(ctx)
	if err != nil {
		return nil, err
	}
	return getExistingProject(ctx, client, projectCode)
}

// EnsureProject 确保给定 code 的 BCS 项目存在。
func (m *ProjectManager) EnsureProject(
	ctx context.Context, displayName, projectCode string, bizID int64,
) (*cloudbcs.Project, error) {
	client, err := m.newClient(ctx)
	if err != nil {
		return nil, err
	}

	project, err := getExistingProject(ctx, client, projectCode)
	if err == nil {
		if bizID > 0 && cast.ToInt64(project.BizID) != bizID {
			return nil, errors.Errorf(
				"existing bcs project(%s) belongs to biz %s, not requested biz %d",
				projectCode,
				project.BizID,
				bizID,
			)
		}
		return project, nil
	}
	if !errors.Is(err, cloudbcs.ErrProjectNotFound) {
		return nil, errors.Wrapf(err, "get bcs project %s before create", projectCode)
	}
	return createProject(ctx, client, displayName, projectCode, bizID)
}

func (m *ProjectManager) newClient(ctx context.Context) (cloudbcs.Client, error) {
	client, err := cloudbcs.New(auth.MustGetUser(ctx))
	if err != nil {
		return nil, errors.Wrap(err, "initial bcs client")
	}
	return client, nil
}

func getExistingProject(ctx context.Context, client cloudbcs.Client, projectCode string) (*cloudbcs.Project, error) {
	project, err := client.GetProject(ctx, projectCode)
	if err != nil {
		return nil, errors.Wrapf(err, "get bcs project %s", projectCode)
	}
	if project == nil || project.ID == "" {
		return nil, errors.New("get bcs project returned empty id")
	}
	if project.Kind != cloudbcs.ProjectKindK8s {
		return nil, errors.Errorf("bcs project(%s) is not a k8s project", projectCode)
	}
	return project, nil
}

func createProject(
	ctx context.Context, client cloudbcs.Client, displayName, projectCode string, bizID int64,
) (*cloudbcs.Project, error) {
	name := displayName
	if name == "" {
		name = projectCode
	}

	in := cloudbcs.CreateProjectInput{
		Name:        name,
		ProjectCode: projectCode,
		Kind:        cloudbcs.ProjectKindK8s,
	}
	if bizID > 0 {
		in.BusinessID = cast.ToString(bizID)
	}

	created, err := client.CreateProject(ctx, in)
	if err != nil {
		return nil, errors.Wrapf(err, "create bcs project %s", projectCode)
	}
	return created, nil
}
