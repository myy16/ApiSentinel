import Fastify, { FastifyInstance } from "fastify";
import cors from "@fastify/cors";
import sensible from "@fastify/sensible";
import { logger } from "./lib/logger.js";
import { authRoutes } from "./modules/auth/auth.routes.js";
import { projectRoutes } from "./modules/project/project.routes.js";
import { endpointRoutes } from "./modules/endpoint/endpoint.routes.js";
import { ingestionRoutes } from "./modules/ingestion/ingestion.routes.js";
import { requestRoutes } from "./modules/request/request.routes.js";

export function buildApp(): FastifyInstance {
  const app = Fastify({
    loggerInstance: logger,
    disableRequestLogging: false,
  });

  // Plugins
  app.register(cors, {
    origin: true,
    credentials: true,
  });

  app.register(sensible);

  // Healthcheck
  app.get("/health", async () => {
    return {
      status: "ok",
      timestamp: new Date().toISOString(),
      service: "apisentinel-backend",
      version: "0.1.0",
    };
  });

  // Public Webhook Gateway (Ingestion)
  app.register(ingestionRoutes, { prefix: "/hook" });

  // Authenticated Management API Routes
  app.register(authRoutes, { prefix: "/api/auth" });
  app.register(projectRoutes, { prefix: "/api/projects" });
  app.register(endpointRoutes, { prefix: "/api" });
  app.register(requestRoutes, { prefix: "/api" });

  // Global error handler (conforming to project.md section 34 Error Model)
  app.setErrorHandler((error, request, reply) => {
    logger.error({ err: error, reqId: request.id }, "Request error");

    const statusCode = error.statusCode || 500;
    const code = (error as any).code || "INTERNAL_ERROR";
    const message = statusCode === 500 ? "Internal Server Error" : error.message;

    reply.status(statusCode).send({
      error: {
        code,
        message,
        requestId: request.id,
      },
    });
  });

  return app;
}
