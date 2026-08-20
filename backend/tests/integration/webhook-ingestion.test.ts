import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { buildApp } from "../../src/app.js";
import { db } from "../../src/lib/db.js";
import { users, organizations, projects, endpoints, capturedRequests } from "../../src/db/schema/schema.js";
import { eq } from "drizzle-orm";
import { EndpointMode } from "@apisentinel/shared";

describe("Webhook Ingestion Gateway & Endpoints Integration Test", () => {
  const app = buildApp();
  const testEmail = `webhook-test-${Date.now()}@apisentinel.dev`;
  let accessToken: string;
  let organizationId: string;
  let projectId: string;
  let passEndpointSlug: string;
  let blockEndpointSlug: string;
  let passEndpointId: string;

  beforeAll(async () => {
    // 1. Register test user
    const authRes = await app.inject({
      method: "POST",
      url: "/api/auth/register",
      payload: {
        email: testEmail,
        password: "Password123!",
        organizationName: "Webhook Ingestion Org",
      },
    });

    const authBody = JSON.parse(authRes.payload);
    accessToken = authBody.accessToken;
    organizationId = authBody.organization.id;

    // 2. Create test project
    const projRes = await app.inject({
      method: "POST",
      url: "/api/projects",
      headers: {
        authorization: `Bearer ${accessToken}`,
        "x-organization-id": organizationId,
      },
      payload: {
        name: "Payment Gateway",
      },
    });

    const projBody = JSON.parse(projRes.payload);
    projectId = projBody.id;
  });

  afterAll(async () => {
    try {
      const user = await db.query.users.findFirst({
        where: eq(users.email, testEmail),
      });
      if (user) {
        await db.delete(users).where(eq(users.id, user.id));
      }
    } catch (e) {
      // Ignore
    }
  });

  it("1. Should create a new webhook endpoint with PASS mode", async () => {
    const slug = `stripe-prod-${Date.now()}`;
    const response = await app.inject({
      method: "POST",
      url: `/api/projects/${projectId}/endpoints`,
      headers: {
        authorization: `Bearer ${accessToken}`,
        "x-organization-id": organizationId,
      },
      payload: {
        name: "Stripe Production Hook",
        slug,
        mode: EndpointMode.PASS,
      },
    });

    expect(response.statusCode).toBe(201);
    const body = JSON.parse(response.payload);
    expect(body.name).toBe("Stripe Production Hook");
    expect(body.slug).toBe(slug);
    expect(body.mode).toBe(EndpointMode.PASS);
    expect(body.isActive).toBe(true);

    passEndpointSlug = slug;
    passEndpointId = body.id;
  });

  it("2. Should capture incoming webhook payload via /hook/:slug", async () => {
    const payload = {
      event: "payment_intent.succeeded",
      amount: 4990,
      currency: "usd",
      customer: {
        email: "customer@example.com",
      },
    };

    const response = await app.inject({
      method: "POST",
      url: `/hook/${passEndpointSlug}`,
      headers: {
        "content-type": "application/json",
        "x-stripe-signature": "t=123456,v1=test_sig",
      },
      payload,
    });

    expect(response.statusCode).toBe(200);
    const body = JSON.parse(response.payload);
    expect(body.success).toBe(true);
    expect(body.requestId).toMatch(/^req_/);
  });

  it("3. Should list captured requests for the endpoint", async () => {
    const response = await app.inject({
      method: "GET",
      url: `/api/endpoints/${passEndpointId}/requests`,
      headers: {
        authorization: `Bearer ${accessToken}`,
        "x-organization-id": organizationId,
      },
    });

    expect(response.statusCode).toBe(200);
    const body = JSON.parse(response.payload);
    expect(body.requests.length).toBeGreaterThanOrEqual(1);

    const first = body.requests[0];
    expect(first.httpMethod).toBe("POST");
    expect(first.headers["x-stripe-signature"]).toBe("t=123456,v1=test_sig");
    expect(first.parsedJson.event).toBe("payment_intent.succeeded");
    expect(first.responseStatus).toBe(200);
  });

  it("4. Should create an endpoint with BLOCK mode and reject webhooks with 403", async () => {
    const slug = `blocked-hook-${Date.now()}`;
    const createRes = await app.inject({
      method: "POST",
      url: `/api/projects/${projectId}/endpoints`,
      headers: {
        authorization: `Bearer ${accessToken}`,
        "x-organization-id": organizationId,
      },
      payload: {
        name: "Blocked Service",
        slug,
        mode: EndpointMode.BLOCK,
      },
    });

    expect(createRes.statusCode).toBe(201);
    blockEndpointSlug = slug;

    // Send webhook to blocked endpoint
    const hookRes = await app.inject({
      method: "POST",
      url: `/hook/${blockEndpointSlug}`,
      payload: { test: "data" },
    });

    expect(hookRes.statusCode).toBe(403);
    const hookBody = JSON.parse(hookRes.payload);
    expect(hookBody.error.code).toBe("POLICY_BLOCKED");
  });

  it("5. Should return 404 for unknown endpoint slug", async () => {
    const response = await app.inject({
      method: "POST",
      url: "/hook/non-existent-slug-xyz",
      payload: {},
    });

    expect(response.statusCode).toBe(404);
    const body = JSON.parse(response.payload);
    expect(body.error.code).toBe("ENDPOINT_NOT_FOUND");
  });
});
