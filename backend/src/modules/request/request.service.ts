import { db } from "../../lib/db.js";
import { capturedRequests, endpoints } from "../../db/schema/schema.js";
import { eq, desc, inArray } from "drizzle-orm";

export class RequestService {
  static async listByEndpoint(endpointId: string, limit = 50, offset = 0) {
    const list = await db.query.capturedRequests.findMany({
      where: eq(capturedRequests.endpointId, endpointId),
      orderBy: [desc(capturedRequests.createdAt)],
      limit,
      offset,
    });

    return list;
  }

  static async listByProject(projectId: string, limit = 50, offset = 0) {
    // Find all endpoints for the project
    const projectEndpoints = await db.query.endpoints.findMany({
      where: eq(endpoints.projectId, projectId),
      columns: { id: true, name: true, slug: true },
    });

    if (projectEndpoints.length === 0) {
      return [];
    }

    const endpointIds = projectEndpoints.map((e) => e.id);
    const endpointMap = new Map(projectEndpoints.map((e) => [e.id, e]));

    const list = await db.query.capturedRequests.findMany({
      where: inArray(capturedRequests.endpointId, endpointIds),
      orderBy: [desc(capturedRequests.createdAt)],
      limit,
      offset,
    });

    return list.map((req) => ({
      ...req,
      endpoint: endpointMap.get(req.endpointId),
    }));
  }

  static async getById(requestId: string) {
    const request = await db.query.capturedRequests.findFirst({
      where: eq(capturedRequests.id, requestId),
    });

    if (!request) {
      const error = new Error("İstek bulunamadı");
      (error as any).statusCode = 404;
      (error as any).code = "REQUEST_NOT_FOUND";
      throw error;
    }

    const endpoint = await db.query.endpoints.findFirst({
      where: eq(endpoints.id, request.endpointId),
    });

    return {
      ...request,
      endpoint,
    };
  }
}
