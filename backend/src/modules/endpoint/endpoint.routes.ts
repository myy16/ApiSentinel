import { FastifyInstance } from "fastify";
import { EndpointController } from "./endpoint.controller.js";
import { authenticate } from "../../middleware/auth.js";
import { requireTenant } from "../../middleware/tenant.js";

export async function endpointRoutes(app: FastifyInstance) {
  app.addHook("preHandler", authenticate);
  app.addHook("preHandler", requireTenant);

  app.get("/projects/:projectId/endpoints", EndpointController.list);
  app.post("/projects/:projectId/endpoints", EndpointController.create);
  app.get("/projects/:projectId/endpoints/:id", EndpointController.getById);
  app.patch("/projects/:projectId/endpoints/:id", EndpointController.update);
  app.delete("/projects/:projectId/endpoints/:id", EndpointController.delete);
}
