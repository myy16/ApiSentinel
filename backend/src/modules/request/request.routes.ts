import { FastifyInstance } from "fastify";
import { RequestController } from "./request.controller.js";
import { authenticate } from "../../middleware/auth.js";
import { requireTenant } from "../../middleware/tenant.js";

export async function requestRoutes(app: FastifyInstance) {
  app.addHook("preHandler", authenticate);
  app.addHook("preHandler", requireTenant);

  app.get("/endpoints/:endpointId/requests", RequestController.listByEndpoint);
  app.get("/projects/:projectId/requests", RequestController.listByProject);
  app.get("/requests/:id", RequestController.getById);
}
