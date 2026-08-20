import { FastifyRequest, FastifyReply } from "fastify";
import { EndpointService } from "./endpoint.service.js";
import { createEndpointSchema, updateEndpointSchema } from "@apisentinel/shared";

export class EndpointController {
  static async list(request: FastifyRequest<{ Params: { projectId: string } }>, reply: FastifyReply) {
    const endpoints = await EndpointService.listEndpoints(request.params.projectId);
    return reply.status(200).send({ endpoints });
  }

  static async create(request: FastifyRequest<{ Params: { projectId: string } }>, reply: FastifyReply) {
    const parseResult = createEndpointSchema.safeParse(request.body);
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

    const endpoint = await EndpointService.createEndpoint(request.params.projectId, parseResult.data);
    return reply.status(201).send(endpoint);
  }

  static async getById(
    request: FastifyRequest<{ Params: { projectId: string; id: string } }>,
    reply: FastifyReply
  ) {
    const endpoint = await EndpointService.getEndpointById(request.params.projectId, request.params.id);
    return reply.status(200).send(endpoint);
  }

  static async update(
    request: FastifyRequest<{ Params: { projectId: string; id: string } }>,
    reply: FastifyReply
  ) {
    const parseResult = updateEndpointSchema.safeParse(request.body);
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

    const endpoint = await EndpointService.updateEndpoint(
      request.params.projectId,
      request.params.id,
      parseResult.data
    );
    return reply.status(200).send(endpoint);
  }

  static async delete(
    request: FastifyRequest<{ Params: { projectId: string; id: string } }>,
    reply: FastifyReply
  ) {
    const result = await EndpointService.deleteEndpoint(request.params.projectId, request.params.id);
    return reply.status(200).send(result);
  }
}
