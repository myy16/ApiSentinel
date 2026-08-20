import { pgTable, uuid, varchar, text, boolean, timestamp, jsonb, integer, inet } from "drizzle-orm/pg-core";
import { EndpointMode } from "@apisentinel/shared";
import { projects } from "./users.js";

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
