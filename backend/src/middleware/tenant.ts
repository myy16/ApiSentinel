import { FastifyRequest, FastifyReply } from "fastify";
import { db } from "../lib/db.js";
import { memberships } from "../db/schema/users.js";
import { eq, and } from "drizzle-orm";
import { UserRole } from "@apisentinel/shared";

declare module "fastify" {
  interface FastifyRequest {
    organizationId?: string;
    organizationRole?: UserRole;
  }
}

export async function requireTenant(request: FastifyRequest, reply: FastifyReply) {
  if (!request.user) {
    return reply.status(401).send({
      error: {
        code: "AUTH_REQUIRED",
        message: "Authentication required",
        requestId: request.id,
      },
    });
  }

  // Organization can come from header 'x-organization-id', query, or token payload
  const orgIdHeader = request.headers["x-organization-id"] as string | undefined;
  const targetOrgId = orgIdHeader || request.user.organizationId;

  if (!targetOrgId) {
    return reply.status(400).send({
      error: {
        code: "TENANT_REQUIRED",
        message: "Organization ID is required (via token or 'x-organization-id' header)",
        requestId: request.id,
      },
    });
  }

  // Check if user has active membership in this organization
  const membership = await db.query.memberships.findFirst({
    where: and(
      eq(memberships.userId, request.user.userId),
      eq(memberships.organizationId, targetOrgId)
    ),
  });

  if (!membership) {
    return reply.status(403).send({
      error: {
        code: "FORBIDDEN",
        message: "Access to this organization is denied",
        requestId: request.id,
      },
    });
  }

  request.organizationId = targetOrgId;
  request.organizationRole = membership.role as UserRole;
}
