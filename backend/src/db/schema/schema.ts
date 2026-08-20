import { pgTable, uuid, varchar, text, boolean, timestamp, jsonb, integer, inet, unique } from "drizzle-orm/pg-core";
import { UserRole, EndpointMode, Severity, PolicyAction, FindingCategory, ReplayStatus, ReplayTargetType, AgentStatus } from "@apisentinel/shared";

// 1. Users & Organizations (Phase 1)
export const users = pgTable("users", {
  id: uuid("id").defaultRandom().primaryKey(),
  email: varchar("email", { length: 255 }).notNull().unique(),
  passwordHash: text("password_hash").notNull(),
  createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
});

export const organizations = pgTable("organizations", {
  id: uuid("id").defaultRandom().primaryKey(),
  name: varchar("name", { length: 150 }).notNull(),
  createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
});

export const memberships = pgTable(
  "memberships",
  {
    id: uuid("id").defaultRandom().primaryKey(),
    organizationId: uuid("organization_id")
      .notNull()
      .references(() => organizations.id, { onDelete: "cascade" }),
    userId: uuid("user_id")
      .notNull()
      .references(() => users.id, { onDelete: "cascade" }),
    role: varchar("role", { length: 50 }).notNull().default(UserRole.OWNER),
    createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
  },
  (table) => ({
    orgUserUnique: unique("org_user_unique").on(table.organizationId, table.userId),
  })
);

export const projects = pgTable("projects", {
  id: uuid("id").defaultRandom().primaryKey(),
  organizationId: uuid("organization_id")
    .notNull()
    .references(() => organizations.id, { onDelete: "cascade" }),
  name: varchar("name", { length: 150 }).notNull(),
  createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
});

// 2. Endpoints & Requests (Phase 2)
export const endpoints = pgTable("endpoints", {
  id: uuid("id").defaultRandom().primaryKey(),
  projectId: uuid("project_id")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
  slug: varchar("slug", { length: 128 }).notNull().unique(),
  name: varchar("name", { length: 100 }).notNull(),
  mode: varchar("mode", { length: 30 }).notNull().default(EndpointMode.PASS),
  isActive: boolean("is_active").notNull().default(true),
  upstreamUrl: text("upstream_url"),
  createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
});

export const capturedRequests = pgTable("captured_requests", {
  id: uuid("id").defaultRandom().primaryKey(),
  endpointId: uuid("endpoint_id")
    .notNull()
    .references(() => endpoints.id, { onDelete: "cascade" }),
  requestId: varchar("request_id", { length: 100 }).notNull().unique(),
  httpMethod: varchar("http_method", { length: 10 }).notNull(),
  headers: jsonb("headers").notNull(),
  queryParams: jsonb("query_params").notNull(),
  rawBody: text("raw_body"),
  maskedBody: text("masked_body"),
  parsedJson: jsonb("parsed_json"),
  clientIp: inet("client_ip"),
  responseStatus: integer("response_status"),
  processingStatus: varchar("processing_status", { length: 30 }).default("RECEIVED"),
  createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
});

// 3. Security Findings & Rules (Phase 4 & 5)
export const rules = pgTable("rules", {
  id: uuid("id").defaultRandom().primaryKey(),
  name: varchar("name", { length: 150 }).notNull(),
  category: varchar("category", { length: 50 }).notNull(),
  ruleType: varchar("rule_type", { length: 100 }).notNull(),
  severity: varchar("severity", { length: 20 }).notNull().default(Severity.HIGH),
  enabled: boolean("enabled").notNull().default(true),
  configuration: jsonb("configuration"),
});

export const securityFindings = pgTable("security_findings", {
  id: uuid("id").defaultRandom().primaryKey(),
  requestId: uuid("request_id")
    .notNull()
    .references(() => capturedRequests.id, { onDelete: "cascade" }),
  ruleId: uuid("rule_id").references(() => rules.id, { onDelete: "set null" }),
  category: varchar("category", { length: 50 }).notNull(),
  type: varchar("type", { length: 100 }).notNull(),
  severity: varchar("severity", { length: 20 }).notNull(),
  action: varchar("action", { length: 20 }).notNull(),
  fieldPath: text("field_path"),
  message: text("message").notNull(),
  evidenceMasked: text("evidence_masked"),
  createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
});

export const policies = pgTable("policies", {
  id: uuid("id").defaultRandom().primaryKey(),
  projectId: uuid("project_id")
    .notNull()
    .references(() => projects.id, { onDelete: "cascade" }),
  name: varchar("name", { length: 150 }).notNull(),
  configuration: jsonb("configuration").notNull(),
  createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
});

// 4. Mock & Replay (Phase 6 & 7)
export const mockRules = pgTable("mock_rules", {
  id: uuid("id").defaultRandom().primaryKey(),
  endpointId: uuid("endpoint_id")
    .notNull()
    .references(() => endpoints.id, { onDelete: "cascade" }),
  name: varchar("name", { length: 150 }).notNull(),
  condition: jsonb("condition"),
  statusCode: integer("status_code").notNull().default(200),
  delayMs: integer("delay_ms").notNull().default(0),
  responseHeaders: jsonb("response_headers"),
  responseBody: jsonb("response_body"),
  enabled: boolean("enabled").notNull().default(true),
});

export const replayJobs = pgTable("replay_jobs", {
  id: uuid("id").defaultRandom().primaryKey(),
  sourceRequestId: uuid("source_request_id")
    .notNull()
    .references(() => capturedRequests.id, { onDelete: "cascade" }),
  targetType: varchar("target_type", { length: 30 }).notNull(),
  targetUrl: text("target_url"),
  status: varchar("status", { length: 30 }).notNull().default(ReplayStatus.PENDING),
  responseStatus: integer("response_status"),
  responseBody: text("response_body"),
  createdAt: timestamp("created_at", { withTimezone: true }).defaultNow().notNull(),
  completedAt: timestamp("completed_at", { withTimezone: true }),
});
