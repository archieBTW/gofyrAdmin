import type {
  AdminUser,
  SchemaResponse,
  MediaFile,
  AuditLog,
  RouteInfo,
  WorkbenchExecuteResponse,
} from './types';

declare global {
  interface Window {
    __GOFYR_CONFIG__?: {
      basePath?: string;
      title?: string;
    };
  }
}

export function getBasePath(): string {
  if (typeof window !== 'undefined' && window.__GOFYR_CONFIG__?.basePath) {
    return window.__GOFYR_CONFIG__.basePath.replace(/\/+$/, '');
  }
  // Try to infer from pathname if mounted at /something
  if (typeof window !== 'undefined') {
    const path = window.location.pathname;
    if (path.includes('/admin')) {
      const idx = path.indexOf('/admin');
      return path.substring(0, idx + 6);
    }
  }
  return '/admin';
}

export function getToken(): string | null {
  if (typeof window !== 'undefined') {
    return localStorage.getItem('gofyr_token');
  }
  return null;
}

export function setToken(token: string): void {
  if (typeof window !== 'undefined') {
    localStorage.setItem('gofyr_token', token);
  }
}

export function removeToken(): void {
  if (typeof window !== 'undefined') {
    localStorage.removeItem('gofyr_token');
  }
}

async function request<T>(
  endpoint: string,
  options: RequestInit = {}
): Promise<T> {
  const basePath = getBasePath();
  const url = `${basePath}/api${endpoint.startsWith('/') ? endpoint : `/${endpoint}`}`;

  const headers = new Headers(options.headers || {});
  const token = getToken();
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`);
  }

  // Set default JSON Content-Type if body is not FormData
  if (
    options.body &&
    !(options.body instanceof FormData) &&
    !headers.has('Content-Type')
  ) {
    headers.set('Content-Type', 'application/json');
  }

  const response = await fetch(url, {
    ...options,
    headers,
  });

  if (response.status === 401 && !endpoint.includes('/auth/')) {
    removeToken();
    window.location.reload();
    throw new Error('Session expired. Please sign in again.');
  }

  let data: any;
  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/json')) {
    data = await response.json();
  } else {
    data = await response.text();
  }

  if (!response.ok) {
    const errorMsg =
      (data && typeof data === 'object' && data.error) ||
      (typeof data === 'string' && data) ||
      `Request failed with status ${response.status}`;
    throw new Error(errorMsg);
  }

  return data as T;
}

export const api = {
  // Auth
  async login(email: string, password: string) {
    return request<{
      pre_auth_token: string;
      mfa_enrolled: boolean;
      user: AdminUser;
    }>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
  },

  async setupMFA(preAuthToken: string) {
    return request<{
      secret: string;
      provisioning_uri: string;
      qr_code_base64: string;
      recovery_codes: string[];
    }>('/auth/mfa/setup', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${preAuthToken}`,
      },
    });
  },

  async verifyMFA(preAuthToken: string, code: string) {
    return request<{
      token: string;
      user: AdminUser;
    }>('/auth/mfa/verify', {
      method: 'POST',
      body: JSON.stringify({ pre_auth_token: preAuthToken, code }),
    });
  },

  async getMe() {
    return request<{ user: AdminUser }>('/auth/me');
  },

  async logout() {
    try {
      await request('/auth/logout', { method: 'POST' });
    } finally {
      removeToken();
    }
  },

  // Schema
  async getSchema() {
    return request<SchemaResponse>('/schema');
  },

  // CRUD
  async listRecords(
    datasource: string,
    model: string,
    params: {
      page?: number;
      limit?: number;
      search?: string;
      sort?: string;
      filters?: Record<string, string>;
    } = {}
  ) {
    const query = new URLSearchParams();
    if (params.page) query.set('page', params.page.toString());
    if (params.limit) query.set('limit', params.limit.toString());
    if (params.search) query.set('search', params.search);
    if (params.sort) query.set('sort', params.sort);
    if (params.filters) {
      for (const [k, v] of Object.entries(params.filters)) {
        if (v) query.set(`filter[${k}]`, v);
      }
    }

    const qs = query.toString();
    return request<{
      data: Record<string, any>[];
      total: number;
      page: number;
      limit: number;
      total_pages: number;
    }>(`/ds/${datasource}/records/${model}${qs ? `?${qs}` : ''}`);
  },

  async getRecord(datasource: string, model: string, id: string | number) {
    return request<{ data: Record<string, any> }>(
      `/ds/${datasource}/records/${model}/${id}`
    );
  },

  async createRecord(
    datasource: string,
    model: string,
    data: Record<string, any>
  ) {
    return request<{ data: Record<string, any> }>(
      `/ds/${datasource}/records/${model}`,
      {
        method: 'POST',
        body: JSON.stringify(data),
      }
    );
  },

  async updateRecord(
    datasource: string,
    model: string,
    id: string | number,
    data: Record<string, any>
  ) {
    return request<{ data: Record<string, any> }>(
      `/ds/${datasource}/records/${model}/${id}`,
      {
        method: 'PUT',
        body: JSON.stringify(data),
      }
    );
  },

  async deleteRecord(datasource: string, model: string, id: string | number) {
    return request<{ message: string }>(
      `/ds/${datasource}/records/${model}/${id}`,
      {
        method: 'DELETE',
      }
    );
  },

  async bulkDeleteRecords(
    datasource: string,
    model: string,
    ids: (string | number)[]
  ) {
    return request<{ message: string; count: number }>(
      `/ds/${datasource}/records/${model}/bulk-delete`,
      {
        method: 'POST',
        body: JSON.stringify({ ids }),
      }
    );
  },

  // Media
  async uploadMedia(
    file: File,
    disk?: string,
    folder?: string,
    onProgress?: (percent: number) => void
  ): Promise<{ data: MediaFile }> {
    const basePath = getBasePath();
    const url = `${basePath}/api/media/upload`;

    const formData = new FormData();
    formData.append('file', file);
    if (disk) formData.append('disk', disk);
    if (folder) formData.append('folder', folder);

    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open('POST', url);

      const token = getToken();
      if (token) {
        xhr.setRequestHeader('Authorization', `Bearer ${token}`);
      }

      if (xhr.upload && onProgress) {
        xhr.upload.onprogress = (event) => {
          if (event.lengthComputable) {
            const percent = Math.round((event.loaded / event.total) * 100);
            onProgress(percent);
          }
        };
      }

      xhr.onload = () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          try {
            resolve(JSON.parse(xhr.responseText));
          } catch (e) {
            reject(new Error('Invalid JSON response'));
          }
        } else {
          try {
            const err = JSON.parse(xhr.responseText);
            reject(new Error(err.error || `Upload failed with status ${xhr.status}`));
          } catch {
            reject(new Error(`Upload failed with status ${xhr.status}`));
          }
        }
      };

      xhr.onerror = () => reject(new Error('Network error during upload'));
      xhr.send(formData);
    });
  },

  async listMediaFiles(params: {
    page?: number;
    limit?: number;
    disk?: string;
    search?: string;
    type?: string;
  } = {}) {
    const query = new URLSearchParams();
    if (params.page) query.set('page', params.page.toString());
    if (params.limit) query.set('limit', params.limit.toString());
    if (params.disk) query.set('disk', params.disk);
    if (params.search) query.set('search', params.search);
    if (params.type) query.set('type', params.type);

    const qs = query.toString();
    return request<{
      data: MediaFile[];
      total: number;
      page: number;
      limit: number;
      total_pages: number;
    }>(`/media/files${qs ? `?${qs}` : ''}`);
  },

  async deleteMediaFile(id: number) {
    return request<{ message: string }>(`/media/files/${id}`, {
      method: 'DELETE',
    });
  },

  // Workbench
  async listWorkbenchRoutes() {
    return request<{ routes: RouteInfo[]; count: number }>('/workbench/routes');
  },

  async executeWorkbenchRoute(data: {
    method: string;
    path: string;
    query_params?: Record<string, string>;
    headers?: Record<string, string>;
    body?: string;
  }) {
    return request<WorkbenchExecuteResponse>('/workbench/execute', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  // Dynamic Schema & Models
  async createCustomModel(data: {
    datasource: string;
    name: string;
    table_name: string;
    fields: any[];
  }) {
    return request<{ message: string; model: any }>('/schema/models', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  // Custom Routes
  async createCustomRoute(data: {
    method: string;
    path: string;
    status_code?: number;
    headers_json?: string;
    response_body?: string;
    description?: string;
  }) {
    return request<{ message: string; route: any }>('/workbench/custom-routes', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  },

  async deleteCustomRoute(id: number) {
    return request<{ message: string }>(`/workbench/custom-routes/${id}`, {
      method: 'DELETE',
    });
  },

  // Audit Logs
  async listAuditLogs(params: {
    page?: number;
    limit?: number;
    search?: string;
    action?: string;
  } = {}) {
    const query = new URLSearchParams();
    if (params.page) query.set('page', params.page.toString());
    if (params.limit) query.set('limit', params.limit.toString());
    if (params.search) query.set('search', params.search);
    if (params.action) query.set('action', params.action);

    const qs = query.toString();
    return request<{
      data: AuditLog[];
      total: number;
      page: number;
      limit: number;
      total_pages: number;
    }>(`/audit-logs${qs ? `?${qs}` : ''}`);
  },
};
