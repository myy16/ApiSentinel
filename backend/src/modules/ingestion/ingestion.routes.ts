import { FastifyInstance } from "fastify";
import { IngestionController } from "./ingestion.controller.js";

export async function ingestionRoutes(app: FastifyInstance) {
  // Public dynamic webhook ingestion endpoint (accepts all HTTP methods)
  app.all("/:slug", IngestionController.handleWebhook);
}
