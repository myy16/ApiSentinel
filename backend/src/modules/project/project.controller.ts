import { FastifyRequest, FastifyReply } from "fastify";
import { ProjectService } from "./project.service.js";
import { createProjectSchema } from "@apisentinel/shared";

export class ProjectController {
  static async list(request: FastifyRequest, reply: FastifyReply) {
    const orgId = request.organizationId!;
    const projects = await ProjectService.listProjects(orgId);
    return reply.status(200).send({ projects });
  }

  static async create(request: FastifyRequest, reply: FastifyReply) {
    const parseResult = createProjectSchema.safeParse(request.body);
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

    const orgId = request.organizationId!;
    const project = await ProjectService.createProject(orgId, parseResult.data);
    return reply.status(201).send(project);
  }

  static async getById(request: FastifyRequest<{ Params: { id: string } }>, reply: FastifyReply) {
    const orgId = request.organizationId!;
    const project = await ProjectService.getProjectById(orgId, request.params.id);
    return reply.status(200).send(project);
  }

  static async update(request: FastifyRequest<{ Params: { id: string } }>, reply: FastifyReply) {
    const parseResult = createProjectSchema.partial().safeParse(request.body);
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

    const orgId = request.organizationId!;
    const project = await ProjectService.updateProject(orgId, request.params.id, parseResult.data);
    return reply.status(200).send(project);
  }

  static async delete(request: FastifyRequest<{ Params: { id: string } }>, reply: FastifyReply) {
    const orgId = request.organizationId!;
    const result = await ProjectService.deleteProject(orgId, request.params.id);
    return reply.status(200).send(result);
  }
}
