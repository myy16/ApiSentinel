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
} from "../constants";

export interface User {
  id: string;
  email: string;
  createdAt: Date;
}

export interface Organization {
  id: string;
  name: string;
  role?: string;
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
  requestId?: string | null;
  projectId?: string | null;
  scanId?: string | null;
  sourceType?: string | null; // "WEBHOOK" | "AGENT_GIT"
  ruleId?: string | null;
  category: FindingCategory;
  type: FindingType | string;
  severity: Severity;
  action: PolicyAction;
  fieldPath?: string | null;
  filePath?: string | null;
  lineNumber?: number | null;
  commitHash?: string | null;
  repository?: string | null;
  message: string;
  evidenceMasked?: string | null;
  confidence?: number | null;
  createdAt: Date;
}

export interface AgentScan {
  id: string;
  projectId: string;
  agentId: string;
  repository: string;
  branch: string;
  commitHash: string;
  scanType: string;
  totalFindings: number;
  action: string;
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

export type PayloadMode = "REDACTED" | "RAW";

export interface ForwardingConfig {
  id: string;
  endpointId: string;
  targetUrl: string;
  maxRetries: number;
  timeoutMs: number;
  customHeaders: Record<string, string>;
  isEnabled: boolean;
  payloadMode: PayloadMode;
  createdAt: Date;
}

export interface ForwardingDlq {
  id: string;
  endpointId: string;
  requestId: string;
  targetUrl: string;
  attempts: number;
  maxRetries: number;
  lastError?: string | null;
  payload?: string | null;
  payloadMode: PayloadMode;
  status: "PENDING" | "PROCESSING" | "RETRY_WAIT" | "SENT" | "DLQ";
  lockedAt?: Date | null;
  lockedBy?: string | null;
  nextRetryAt: Date;
  createdAt: Date;
  lastAttemptAt: Date;
}

export type RequestState = "RECEIVED" | "VERIFIED" | "ACCEPTED" | "REJECTED_SIGNATURE" | "BLOCKED_POLICY";

export type DeliveryState = "NOT_CONFIGURED" | "PENDING" | "PROCESSING" | "RETRY_WAIT" | "DELIVERED" | "DEAD_LETTER";

export interface DeliveryJob {
  id: string;
  endpointId: string;
  requestId: string;
  targetUrl: string;
  status: DeliveryState;
  attempts: number;
  maxRetries: number;
  nextRetryAt: string;
  lockedAt?: string | null;
  lockedBy?: string | null;
  idempotencyKey?: string | null;
  lastError?: string | null;
  payloadMode: PayloadMode;
  createdAt: string;
  updatedAt: string;
  completedAt?: string | null;
}

export interface DeliveryAttempt {
  id: string;
  jobId: string;
  attemptNumber: number;
  startedAt: string;
  finishedAt: string;
  latencyMs: number;
  responseStatusCode?: number | null;
  errorType?: string | null;
  errorMessage?: string | null;
  requestHeadersSent: Record<string, string>;
  responseHeadersReceived: Record<string, string>;
  responseBodySnippet?: string | null;
  createdAt: string;
}

export interface DeliveryTimelineStep {
  step: "INGEST" | "SECURITY_INSPECTION" | "ATTEMPT";
  attempt?: number;
  status: "COMPLETED" | "SUCCESS" | "FAILED" | "PENDING";
  statusCode?: number;
  latencyMs?: number;
  error?: string;
  startedAt?: string;
  finishedAt?: string;
  timestamp?: string;
  description: string;
}

export interface DeliveryTimelineData {
  job: DeliveryJob;
  request: CapturedRequest;
  attempts: DeliveryAttempt[];
  timeline: DeliveryTimelineStep[];
}

export interface DeliveryKPIs {
  totalDeliveries: number;
  delivered: number;
  deadLetter: number;
  pending: number;
  retryWait: number;
  successRate: number;
  dlqBacklog: number;
  timestamp: string;
}

export interface AuditLog {
  id: string;
  organizationId: string;
  projectId?: string | null;
  userId?: string | null;
  action: string;
  resourceType: string;
  resourceId: string;
  justification?: string | null;
  ipAddress?: string | null;
  metadata?: Record<string, unknown>;
  createdAt: string;
}


