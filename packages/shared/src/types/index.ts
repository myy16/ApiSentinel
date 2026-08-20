import {
  UserRole,
  EndpointMode,
  Severity,
  PolicyAction,
  FindingCategory,
  FindingType,
  ReplayStatus,
  ReplayTargetType,
  AgentStatus,
} from "../constants/index";

export interface User {
  id: string;
  email: string;
  createdAt: Date;
}

export interface Organization {
  id: string;
  name: string;
  createdAt: Date;
}

export interface Membership {
  id: string;
  organizationId: string;
  userId: string;
  role: UserRole;
  createdAt: Date;
}

export interface Project {
  id: string;
  organizationId: string;
  name: string;
  createdAt: Date;
}

export interface Endpoint {
  id: string;
  projectId: string;
  slug: string;
  name: string;
  mode: EndpointMode;
  isActive: boolean;
  upstreamUrl?: string | null;
  createdAt: Date;
}

export interface CapturedRequest {
  id: string;
  endpointId: string;
  requestId: string;
  httpMethod: string;
  headers: Record<string, string | string[] | undefined>;
  queryParams: Record<string, string | string[] | undefined>;
  rawBody?: string | null;
  maskedBody?: string | null;
  parsedJson?: unknown | null;
  clientIp?: string | null;
  responseStatus?: number | null;
  processingStatus?: string | null;
  createdAt: Date;
}

export interface SecurityFinding {
  id: string;
  requestId: string;
  ruleId?: string | null;
  category: FindingCategory;
  type: FindingType | string;
  severity: Severity;
  action: PolicyAction;
  fieldPath?: string | null;
  message: string;
  evidenceMasked?: string | null;
  confidence?: number | null;
  createdAt: Date;
}

export interface PolicyRule {
  category?: FindingCategory;
  type?: FindingType | string;
  severity?: Severity;
  action: PolicyAction;
}

export interface PolicyConfig {
  rules: PolicyRule[];
}

export interface MockRule {
  id: string;
  endpointId: string;
  name: string;
  condition?: Record<string, unknown> | null;
  statusCode: number;
  delayMs: number;
  responseHeaders?: Record<string, string> | null;
  responseBody?: unknown | null;
  enabled: boolean;
}

export interface ReplayJob {
  id: string;
  sourceRequestId: string;
  targetType: ReplayTargetType;
  targetUrl?: string | null;
  status: ReplayStatus;
  responseStatus?: number | null;
  responseBody?: string | null;
  createdAt: Date;
  completedAt?: Date | null;
}

export interface Agent {
  id: string;
  userId: string;
  name: string;
  status: AgentStatus;
  lastSeenAt?: Date | null;
  createdAt: Date;
}
