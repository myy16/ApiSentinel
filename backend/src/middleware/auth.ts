import { FastifyRequest, FastifyReply } from "fastify";
import { verifyToken, TokenPayload } from "../lib/jwt.js";
import { UserRole } from "@apisentinel/shared";

declare module "fastify" {
  interface FastifyRequest {
    user?: TokenPayload;
  }
}

export async function authenticate(request: FastifyRequest, reply: FastifyReply) {
  const authHeader = request.headers.authorization;

  if (!authHeader || !authHeader.startsWith("Bearer ")) {
    return reply.status(401).send({
      error: {
        code: "AUTH_REQUIRED",
        message: "Authorization token is required (Bearer <token>)",
        requestId: request.id,
      },
    });
  }

  const token = authHeader.substring(7);

  try {
    const payload = await verifyToken(token, "access");
    request.user = payload;
  } catch (err) {
    return reply.status(401).send({
      error: {
        code: "AUTH_INVALID",
        message: "Invalid or expired authorization token",
        requestId: request.id,
      },
    });
  }
}

export function requireRoles(...allowedRoles: UserRole[]) {
  return async (request: FastifyRequest, reply: FastifyReply) => {
    if (!request.user) {
      return reply.status(401).send({
        error: {
          code: "AUTH_REQUIRED",
          message: "Authentication required",
          requestId: request.id,
        },
      });
    }

    if (request.user.role && !allowedRoles.includes(request.user.role)) {
      return reply.status(403).send({
        error: {
          code: "FORBIDDEN",
          message: "Insufficient permissions for this action",
          requestId: request.id,
        },
      });
    }
  };
}
