export const UserRole = {
  OWNER: "OWNER",
  ADMIN: "ADMIN",
  DEVELOPER: "DEVELOPER",
  VIEWER: "VIEWER",
} as const;

export type UserRole = (typeof UserRole)[keyof typeof UserRole];

export const EndpointMode = {
  PASS: "PASS",
  BLOCK: "BLOCK",
  MOCK: "MOCK",
  CAPTURE_ONLY: "CAPTURE_ONLY",
} as const;

export type EndpointMode = (typeof EndpointMode)[keyof typeof EndpointMode];

export const Severity = {
  CRITICAL: "CRITICAL",
  HIGH: "HIGH",
  MEDIUM: "MEDIUM",
  LOW: "LOW",
  INFO: "INFO",
} as const;

export type Severity = (typeof Severity)[keyof typeof Severity];

export const PolicyAction = {
  ALLOW: "ALLOW",
  WARN: "WARN",
  ALERT: "ALERT",
  MASK: "MASK",
  BLOCK: "BLOCK",
} as const;

export type PolicyAction = (typeof PolicyAction)[keyof typeof PolicyAction];

export const FindingCategory = {
  PII: "PII",
  SECRET: "SECRET",
  INJECTION: "INJECTION",
  AUTH: "AUTH",
  SCHEMA: "SCHEMA",
  DUPLICATE: "DUPLICATE",
  DATA_QUALITY: "DATA_QUALITY",
} as const;

export type FindingCategory = (typeof FindingCategory)[keyof typeof FindingCategory];

export const FindingType = {
  // PII
  EMAIL: "EMAIL",
  PHONE: "PHONE",
  TCKN: "TCKN",
  CREDIT_CARD: "CREDIT_CARD",
  IBAN: "IBAN",
  // SECRET
  API_KEY: "API_KEY",
  AWS_KEY: "AWS_KEY",
  GITHUB_TOKEN: "GITHUB_TOKEN",
  JWT_EXPOSURE: "JWT_EXPOSURE",
  PRIVATE_KEY: "PRIVATE_KEY",
  DB_PASSWORD: "DB_PASSWORD",
  // INJECTION
  SQL_INJECTION: "SQL_INJECTION",
  XSS: "XSS",
  // DUPLICATE
  DUPLICATE_EVENT: "DUPLICATE_EVENT",
  // SCHEMA
  SCHEMA_VIOLATION: "SCHEMA_VIOLATION",
} as const;

export type FindingType = (typeof FindingType)[keyof typeof FindingType];

export const ReplayStatus = {
  PENDING: "PENDING",
  RUNNING: "RUNNING",
  COMPLETED: "COMPLETED",
  FAILED: "FAILED",
} as const;

export type ReplayStatus = (typeof ReplayStatus)[keyof typeof ReplayStatus];

export const ReplayTargetType = {
  PUBLIC_URL: "PUBLIC_URL",
  PROJECT_ENDPOINT: "PROJECT_ENDPOINT",
  LOCAL_AGENT: "LOCAL_AGENT",
} as const;

export type ReplayTargetType = (typeof ReplayTargetType)[keyof typeof ReplayTargetType];

export const AgentStatus = {
  ONLINE: "ONLINE",
  OFFLINE: "OFFLINE",
  REVOKED: "REVOKED",
} as const;

export type AgentStatus = (typeof AgentStatus)[keyof typeof AgentStatus];
