import { SignJWT, jwtVerify } from "jose";
import { config } from "./config.js";
import { UserRole } from "@apisentinel/shared";

const secretKey = new TextEncoder().encode(config.JWT_SECRET);

export interface TokenPayload {
  userId: string;
  email: string;
  organizationId?: string;
  role?: UserRole;
}

export async function generateAccessToken(payload: TokenPayload): Promise<string> {
  return new SignJWT({ ...payload, type: "access" })
    .setProtectedHeader({ alg: "HS256" })
    .setIssuedAt()
    .setExpirationTime(config.JWT_EXPIRES_IN)
    .sign(secretKey);
}

export async function generateRefreshToken(payload: TokenPayload): Promise<string> {
  return new SignJWT({ ...payload, type: "refresh" })
    .setProtectedHeader({ alg: "HS256" })
    .setIssuedAt()
    .setExpirationTime(config.REFRESH_TOKEN_EXPIRES_IN)
    .sign(secretKey);
}

export async function verifyToken(token: string, expectedType: "access" | "refresh" = "access"): Promise<TokenPayload> {
  const { payload } = await jwtVerify(token, secretKey);

  if (payload.type !== expectedType) {
    throw new Error(`Invalid token type: expected ${expectedType}`);
  }

  return {
    userId: payload.userId as string,
    email: payload.email as string,
    organizationId: payload.organizationId as string | undefined,
    role: payload.role as UserRole | undefined,
  };
}
