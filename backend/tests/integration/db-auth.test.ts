import { describe, it, expect, beforeAll, afterAll } from "vitest";
import { buildApp } from "../../src/app.js";
import { db } from "../../src/lib/db.js";
import { users, organizations, projects } from "../../src/db/schema/users.js";
import { eq } from "drizzle-orm";

describe("Live PostgreSQL Database & Auth End-to-End Test", () => {
  const app = buildApp();
  const testEmail = `test-${Date.now()}@apisentinel.dev`;
  const testPassword = "SuperSecretPassword123!";
  let accessToken: string;
  let organizationId: string;
  let createdProjectId: string;

  afterAll(async () => {
    // Cleanup test data
    try {
      const user = await db.query.users.findFirst({
        where: eq(users.email, testEmail),
      });
      if (user) {
        await db.delete(users).where(eq(users.id, user.id));
      }
    } catch (e) {
      // Ignore cleanup error
    }
  });

  it("1. Should register a new user with default organization in live Postgres", async () => {
    const response = await app.inject({
      method: "POST",
      url: "/api/auth/register",
      payload: {
        email: testEmail,
        password: testPassword,
        organizationName: "E2E Test Org",
      },
    });

    expect(response.statusCode).toBe(201);
    const body = JSON.parse(response.payload);
    expect(body.user.email).toBe(testEmail);
    expect(body.organization.name).toBe("E2E Test Org");
    expect(body.accessToken).toBeDefined();
    expect(body.refreshToken).toBeDefined();

    accessToken = body.accessToken;
    organizationId = body.organization.id;
  });

  it("2. Should reject duplicate email registration", async () => {
    const response = await app.inject({
      method: "POST",
      url: "/api/auth/register",
      payload: {
        email: testEmail,
        password: testPassword,
      },
    });

    expect(response.statusCode).toBe(409);
    const body = JSON.parse(response.payload);
    expect(body.error.code).toBe("EMAIL_EXISTS");
  });

  it("3. Should login with correct credentials and receive fresh tokens", async () => {
    const response = await app.inject({
      method: "POST",
      url: "/api/auth/login",
      payload: {
        email: testEmail,
        password: testPassword,
      },
    });

    expect(response.statusCode).toBe(200);
    const body = JSON.parse(response.payload);
    expect(body.user.email).toBe(testEmail);
    expect(body.accessToken).toBeDefined();
  });

  it("4. Should fetch current user profile via /api/auth/me", async () => {
    const response = await app.inject({
      method: "GET",
      url: "/api/auth/me",
      headers: {
        authorization: `Bearer ${accessToken}`,
      },
    });

    expect(response.statusCode).toBe(200);
    const body = JSON.parse(response.payload);
    expect(body.user.email).toBe(testEmail);
    expect(body.memberships.length).toBeGreaterThanOrEqual(1);
  });

  it("5. Should create a project in the active organization", async () => {
    const response = await app.inject({
      method: "POST",
      url: "/api/projects",
      headers: {
        authorization: `Bearer ${accessToken}`,
        "x-organization-id": organizationId,
      },
      payload: {
        name: "Stripe Webhook Service",
      },
    });

    expect(response.statusCode).toBe(201);
    const body = JSON.parse(response.payload);
    expect(body.name).toBe("Stripe Webhook Service");
    expect(body.organizationId).toBe(organizationId);
    createdProjectId = body.id;
  });

  it("6. Should list all projects under the organization", async () => {
    const response = await app.inject({
      method: "GET",
      url: "/api/projects",
      headers: {
        authorization: `Bearer ${accessToken}`,
        "x-organization-id": organizationId,
      },
    });

    expect(response.statusCode).toBe(200);
    const body = JSON.parse(response.payload);
    expect(body.projects.length).toBeGreaterThanOrEqual(1);
    expect(body.projects.some((p: any) => p.name === "Stripe Webhook Service")).toBe(true);
  });

  it("7. Should delete the project", async () => {
    const response = await app.inject({
      method: "DELETE",
      url: `/api/projects/${createdProjectId}`,
      headers: {
        authorization: `Bearer ${accessToken}`,
        "x-organization-id": organizationId,
      },
    });

    expect(response.statusCode).toBe(200);
    const body = JSON.parse(response.payload);
    expect(body.success).toBe(true);
  });
});
