import { FastifyInstance } from "fastify";
import { ProjectController } from "./project.controller.js";
import { authenticate } from "../../middleware/auth.js";
import { requireTenant } from "../../middleware/tenant.js";

export async function projectRoutes(app: FastifyInstance) {
  // All project routes require authentication and tenant context
  app.addHook("preHandler", authenticate);
  app.addHook("preHandler", requireTenant);

  app.get("/", ProjectController.list);
  app.post("/", ProjectController.create);
  app.get("/:id", ProjectController.getById);
  app.patch("/:id", ProjectController.update);
  app.delete("/:id", ProjectController.delete);
}
