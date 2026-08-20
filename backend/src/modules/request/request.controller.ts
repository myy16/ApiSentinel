import { FastifyRequest, FastifyReply } from "fastify";
import { RequestService } from "./request.service.js";

export class RequestController {
  static async listByEndpoint(
    request: FastifyRequest<{ Params: { endpointId: string }; Querystring: { limit?: string; offset?: string } }>,
    reply: FastifyReply
  ) {
    const limit = request.query.limit ? parseInt(request.query.limit, 10) : 50;
    const offset = request.query.offset ? parseInt(request.query.offset, 10) : 0;

    const requests = await RequestService.listByEndpoint(request.params.endpointId, limit, offset);
    return reply.status(200).send({ requests });
  }

  static async listByProject(
    request: FastifyRequest<{ Params: { projectId: string }; Querystring: { limit?: string; offset?: string } }>,
    reply: FastifyReply
  ) {
    const limit = request.query.limit ? parseInt(request.query.limit, 10) : 50;
    const offset = request.query.offset ? parseInt(request.query.offset, 10) : 0;

    const requests = await RequestService.listByProject(request.params.projectId, limit, offset);
    return reply.status(200).send({ requests });
  }

  static async getById(
    request: FastifyRequest<{ Params: { id: string } }>,
    reply: FastifyReply
  ) {
    const req = await RequestService.getById(request.params.id);
    return reply.status(200).send(req);
  }
}
