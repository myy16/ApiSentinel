import { FastifyRequest, FastifyReply } from "fastify";
import { IngestionService } from "./ingestion.service.js";

export class IngestionController {
  static async handleWebhook(
    request: FastifyRequest<{ Params: { slug: string } }>,
    reply: FastifyReply
  ) {
    const result = await IngestionService.processWebhook(request.params.slug, request);
    return reply.status(result.status).send(result.body);
  }
}
