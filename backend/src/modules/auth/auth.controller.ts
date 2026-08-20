import { FastifyRequest, FastifyReply } from "fastify";
import { AuthService } from "./auth.service.js";
import { registerSchema, loginSchema, refreshTokenSchema } from "@apisentinel/shared";

export class AuthController {
  static async register(request: FastifyRequest, reply: FastifyReply) {
    const parseResult = registerSchema.safeParse(request.body);
    if (!parseResult.success) {
      return reply.status(400).send({
        error: {
          code: "VALIDATION_ERROR",
          message: parseResult.error.errors[0].message,
          details: parseResult.error.errors,
          requestId: request.id,
        },
      });
    }

    const result = await AuthService.register(parseResult.data);
    return reply.status(201).send(result);
  }

  static async login(request: FastifyRequest, reply: FastifyReply) {
    const parseResult = loginSchema.safeParse(request.body);
    if (!parseResult.success) {
      return reply.status(400).send({
        error: {
          code: "VALIDATION_ERROR",
          message: parseResult.error.errors[0].message,
          details: parseResult.error.errors,
          requestId: request.id,
        },
      });
    }

    const result = await AuthService.login(parseResult.data);
    return reply.status(200).send(result);
  }

  static async refresh(request: FastifyRequest, reply: FastifyReply) {
    const parseResult = refreshTokenSchema.safeParse(request.body);
    if (!parseResult.success) {
      return reply.status(400).send({
        error: {
          code: "VALIDATION_ERROR",
          message: parseResult.error.errors[0].message,
          details: parseResult.error.errors,
          requestId: request.id,
        },
      });
    }

    const result = await AuthService.refresh(parseResult.data.refreshToken);
    return reply.status(200).send(result);
  }

  static async logout(_request: FastifyRequest, reply: FastifyReply) {
    // Access tokens are stateless at this stage. The client clears both tokens;
    // server-side refresh-token revocation will be introduced with persistent sessions.
    return reply.status(204).send();
  }

  static async me(request: FastifyRequest, reply: FastifyReply) {
    if (!request.user) {
      return reply.status(401).send({
        error: {
          code: "AUTH_REQUIRED",
          message: "Authentication required",
          requestId: request.id,
        },
      });
    }

    const result = await AuthService.getMe(request.user.userId);
    return reply.status(200).send(result);
  }
}
