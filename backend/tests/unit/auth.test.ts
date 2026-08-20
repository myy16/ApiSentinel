import { describe, it, expect } from "vitest";
import { hashPassword, comparePassword } from "../../src/lib/password.js";
import { generateAccessToken, generateRefreshToken, verifyToken } from "../../src/lib/jwt.js";
import { UserRole } from "@apisentinel/shared";

describe("Auth Security & JWT Library", () => {
  it("should securely hash and verify passwords", async () => {
    const password = "SuperSecretPassword123!";
    const hash = await hashPassword(password);

    expect(hash).not.toBe(password);
    expect(await comparePassword(password, hash)).toBe(true);
    expect(await comparePassword("WrongPassword", hash)).toBe(false);
  });

  it("should generate and verify valid access tokens", async () => {
    const payload = {
      userId: "user-uuid-123",
      email: "developer@apisentinel.dev",
      organizationId: "org-uuid-456",
      role: UserRole.OWNER,
    };

    const token = await generateAccessToken(payload);
    expect(typeof token).toBe("string");

    const verified = await verifyToken(token, "access");
    expect(verified.userId).toBe(payload.userId);
    expect(verified.email).toBe(payload.email);
    expect(verified.organizationId).toBe(payload.organizationId);
    expect(verified.role).toBe(UserRole.OWNER);
  });

  it("should reject access verification for refresh tokens", async () => {
    const payload = {
      userId: "user-uuid-123",
      email: "developer@apisentinel.dev",
    };

    const refreshToken = await generateRefreshToken(payload);
    await expect(verifyToken(refreshToken, "access")).rejects.toThrow("Invalid token type");
  });
});
