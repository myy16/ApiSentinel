import { describe, it, expect } from "vitest";
import { buildApp } from "../../src/app.js";
import { generateAccessToken } from "../../src/lib/jwt.js";
import { UserRole } from "@apisentinel/shared";

describe("API Security & Validation Integration Tests", () => {
  const app = buildApp();

  describe("Auth Routes Validation", () => {
    it("should reject registration with invalid email format", async () => {
      const response = await app.inject({
        method: "POST",
        url: "/api/auth/register",
        payload: {
          email: "not-an-email",
          password: "Password123!",
        },
      });

      expect(response.statusCode).toBe(400);
      const body = JSON.parse(response.payload);
      expect(body.error.code).toBe("VALIDATION_ERROR");
    });

    it("should reject registration with short password (< 8 chars)", async () => {
      const response = await app.inject({
        method: "POST",
        url: "/api/auth/register",
        payload: {
          email: "developer@apisentinel.dev",
          password: "short",
        },
      });

      expect(response.statusCode).toBe(400);
      const body = JSON.parse(response.payload);
      expect(body.error.code).toBe("VALIDATION_ERROR");
    });

    it("should reject unauthenticated access to /api/auth/me", async () => {
      const response = await app.inject({
        method: "GET",
        url: "/api/auth/me",
      });

      expect(response.statusCode).toBe(401);
      const body = JSON.parse(response.payload);
      expect(body.error.code).toBe("AUTH_REQUIRED");
    });

    it("should reject requests with invalid bearer token", async () => {
      const response = await app.inject({
        method: "GET",
        url: "/api/auth/me",
        headers: {
          authorization: "Bearer invalid.token.value",
        },
      });

      expect(response.statusCode).toBe(401);
      const body = JSON.parse(response.payload);
      expect(body.error.code).toBe("AUTH_INVALID");
    });
  });

  describe("Project Routes Security & Tenant Isolation", () => {
    it("should require authentication to access /api/projects", async () => {
      const response = await app.inject({
        method: "GET",
        url: "/api/projects",
      });

      expect(response.statusCode).toBe(401);
      const body = JSON.parse(response.payload);
      expect(body.error.code).toBe("AUTH_REQUIRED");
    });

    it("should reject project creation without organization context", async () => {
      const token = await generateAccessToken({
        userId: "user-123",
        email: "user@apisentinel.dev",
      });

      const response = await app.inject({
        method: "POST",
        url: "/api/projects",
        headers: {
          authorization: `Bearer ${token}`,
        },
        payload: {
          name: "Payment API",
        },
      });

      expect(response.statusCode).toBe(400);
      const body = JSON.parse(response.payload);
      expect(body.error.code).toBe("TENANT_REQUIRED");
    });
  });
});
