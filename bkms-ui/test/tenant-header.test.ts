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

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

// interceptors 在模块加载时缓存 window.fetch，必须在导入前替换
const fetchMock = vi.fn(async () => new Response(JSON.stringify({ data: {} }), { status: 200 }));
vi.stubGlobal('fetch', fetchMock);

const { default: Fetch } = await import('../src/api/fetch');
const { setCurrentTenantID, TENANT_ID_HEADER } = await import('../src/api/tenant');

function sentHeaders() {
  const init = fetchMock.mock.calls.at(-1)?.[1] as RequestInit | undefined;
  return (init?.headers ?? {}) as Record<string, string>;
}

describe('tenant request header', () => {
  beforeEach(() => {
    fetchMock.mockClear();
  });

  afterEach(() => {
    setCurrentTenantID('');
  });

  it('adds the current tenant to bkms requests', async () => {
    setCurrentTenantID('tenant-a');

    await new Fetch({ prefix: '' }).get('/apps')();

    expect(sentHeaders()[TENANT_ID_HEADER]).toBe('tenant-a');
  });

  it('omits the header when the tenant is unknown', async () => {
    await new Fetch({ prefix: '' }).get('/apps')();

    expect(sentHeaders()).not.toHaveProperty(TENANT_ID_HEADER);
  });

  it('omits the header for clients that opt out', async () => {
    setCurrentTenantID('tenant-a');

    await new Fetch({ prefix: '', tenantHeader: false }).get('/bcs')();

    expect(sentHeaders()).not.toHaveProperty(TENANT_ID_HEADER);
  });
});
