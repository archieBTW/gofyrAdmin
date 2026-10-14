export interface AdminUser {
  id: number;
  email: string;
  name: string;
  role: string;
  mfa_enabled?: boolean;
  must_reset_password?: boolean;
}

export interface FieldInfo {
  name: string;
  db_name: string;
  type: string;
  gofyr_type: 'text' | 'textarea' | 'number' | 'boolean' | 'datetime' | 'image' | 'gallery' | 'file' | 'json' | 'password' | string;
  disk?: string;
  folder?: string;
  readonly: boolean;
  searchable: boolean;
  hidden: boolean;
  required: boolean;
  label: string;
  is_primary_key: boolean;
}

export interface RelationInfo {
  name: string;
  type: string;
  target_model: string;
  foreign_key?: string;
}

export interface ModelInfo {
  name: string;
  table_name: string;
  primary_key: string;
  fields: FieldInfo[];
  relations: RelationInfo[];
}

export interface DatasourceSchema {
  name: string;
  dialect: 'sqlite' | 'postgres' | 'mysql' | string;
  is_primary: boolean;
  models: ModelInfo[];
}

export interface SchemaResponse {
  title: string;
  base_path: string;
  default_disk: string;
  disks: string[];
  datasources: DatasourceSchema[];
}

export interface MediaFile {
  id: number;
  disk: string;
  folder: string;
  filename: string;
  original_name: string;
  mime_type: string;
  size: number;
  url: string;
  created_at: string;
}

export interface AuditLog {
  id: number;
  user_id: number;
  user_email: string;
  action: string;
  datasource: string;
  model: string;
  record_id: string;
  ip_address: string;
  details: string;
  created_at: string;
}

export interface RouteInfo {
  method: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH' | string;
  path: string;
  handler: string;
  group: string;
  is_admin: boolean;
}

export interface WorkbenchExecuteResponse {
  status_code: number;
  status_text: string;
  duration_ms: number;
  headers: Record<string, string>;
  body: string;
  is_json: boolean;
}
