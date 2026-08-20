import { db } from "../../lib/db.js";
import { endpoints, capturedRequests } from "../../db/schema/schema.js";
import { eq, and, desc, count } from "drizzle-orm";
import { CreateEndpointInput, UpdateEndpointInput, EndpointMode } from "@apisentinel/shared";
import { nanoid } from "nanoid";

export class EndpointService {
  static async listEndpoints(projectId: string) {
    const list = await db.query.endpoints.findMany({
      where: eq(endpoints.projectId, projectId),
      orderBy: [desc(endpoints.createdAt)],
    });

    // Attach captured request counts
    const counts = await db
      .select({
        endpointId: capturedRequests.endpointId,
        count: count(capturedRequests.id),
      })
      .from(capturedRequests)
      .groupBy(capturedRequests.endpointId);

    const countMap = new Map(counts.map((c) => [c.endpointId, Number(c.count)]));

    return list.map((ep) => ({
      ...ep,
      requestCount: countMap.get(ep.id) || 0,
    }));
  }

  static async createEndpoint(projectId: string, input: CreateEndpointInput) {
    const slug = input.slug || `${input.name.toLowerCase().replace(/[^a-z0-9]/g, "-")}-${nanoid(6)}`;

    // Check slug uniqueness
    const existing = await db.query.endpoints.findFirst({
      where: eq(endpoints.slug, slug),
    });

    if (existing) {
      const error = new Error("Bu slug zaten kullanımda");
      (error as any).statusCode = 409;
      (error as any).code = "SLUG_EXISTS";
      throw error;
    }

    const [endpoint] = await db
      .insert(endpoints)
      .values({
        projectId,
        name: input.name,
        slug,
        mode: input.mode || EndpointMode.PASS,
        upstreamUrl: input.upstreamUrl || null,
        isActive: true,
      })
      .returning();

    return endpoint;
  }

  static async getEndpointById(projectId: string, endpointId: string) {
    const endpoint = await db.query.endpoints.findFirst({
      where: and(eq(endpoints.id, endpointId), eq(endpoints.projectId, projectId)),
    });

    if (!endpoint) {
      const error = new Error("Endpoint bulunamadı");
      (error as any).statusCode = 404;
      (error as any).code = "ENDPOINT_NOT_FOUND";
      throw error;
    }

    return endpoint;
  }

  static async getEndpointBySlug(slug: string) {
    const endpoint = await db.query.endpoints.findFirst({
      where: eq(endpoints.slug, slug),
    });

    if (!endpoint) {
      const error = new Error("Endpoint bulunamadı");
      (error as any).statusCode = 404;
      (error as any).code = "ENDPOINT_NOT_FOUND";
      throw error;
    }

    return endpoint;
  }

  static async updateEndpoint(projectId: string, endpointId: string, input: UpdateEndpointInput) {
    await this.getEndpointById(projectId, endpointId);

    const [updated] = await db
      .update(endpoints)
      .set({
        ...(input.name ? { name: input.name } : {}),
        ...(input.mode ? { mode: input.mode } : {}),
        ...(input.isActive !== undefined ? { isActive: input.isActive } : {}),
        ...(input.upstreamUrl !== undefined ? { upstreamUrl: input.upstreamUrl } : {}),
      })
      .where(and(eq(endpoints.id, endpointId), eq(endpoints.projectId, projectId)))
      .returning();

    return updated;
  }

  static async deleteEndpoint(projectId: string, endpointId: string) {
    await this.getEndpointById(projectId, endpointId);

    await db
      .delete(endpoints)
      .where(and(eq(endpoints.id, endpointId), eq(endpoints.projectId, projectId)));

    return { success: true };
  }
}
