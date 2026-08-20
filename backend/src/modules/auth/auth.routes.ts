import { FastifyInstance } from "fastify";
import { AuthController } from "./auth.controller.js";
import { authenticate } from "../../middleware/auth.js";

export async function authRoutes(app: FastifyInstance) {
  app.post("/register", AuthController.register);
  app.post("/login", AuthController.login);
  app.post("/refresh", AuthController.refresh);
  app.get("/me", { preHandler: [authenticate] }, AuthController.me);
}
