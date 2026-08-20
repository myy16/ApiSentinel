import { db } from "../../lib/db.js";
import { endpoints, capturedRequests } from "../../db/schema/schema.js";
import { valkey } from "../../lib/valkey.js";
import { logger } from "../../lib/logger.js";
import { EndpointMode } from "@apisentinel/shared";
import { nanoid } from "nanoid";
import { eq } from "drizzle-orm";
import { FastifyRequest } from "fastify";

export class IngestionService {
  static async processWebhook(slug: string, request: FastifyRequest) {
    const endpoint = await db.query.endpoints.findFirst({
      where: eq(endpoints.slug, slug),
    });

    if (!endpoint) {
      const error = new Error(`Endpoint '${slug}' bulunamadı`);
      (error as any).statusCode = 404;
      (error as any).code = "ENDPOINT_NOT_FOUND";
      throw error;
    }

    if (!endpoint.isActive) {
      const error = new Error(`Endpoint '${slug}' pasif durumda`);
      (error as any).statusCode = 403;
      (error as any).code = "ENDPOINT_DISABLED";
      throw error;
    }

    const requestId = `req_${nanoid(16)}`;
    const httpMethod = request.method;
    const headers = request.headers as Record<string, string | string[] | undefined>;
    const queryParams = (request.query || {}) as Record<string, string | string[] | undefined>;
    const clientIp = request.ip || (request.headers["x-forwarded-for"] as string) || "127.0.0.1";

    let rawBody: string | null = null;
    let parsedJson: unknown = null;

    if (request.body !== undefined && request.body !== null) {
      if (typeof request.body === "string") {
        rawBody = request.body;
        try {
          parsedJson = JSON.parse(request.body);
        } catch {
          // Non-JSON string
        }
      } else {
        parsedJson = request.body;
        rawBody = JSON.stringify(request.body);
      }
    }

    // Determine initial response based on endpoint mode
    let responseStatus = 200;
    let responseBody: any = {
      success: true,
      message: "Webhook accepted",
      requestId,
      timestamp: new Date().toISOString(),
    };

    if (endpoint.mode === EndpointMode.BLOCK) {
      responseStatus = 403;
      responseBody = {
        error: {
          code: "POLICY_BLOCKED",
          message: "Endpoint mode is set to BLOCK",
          requestId,
        },
      };
    }

    // Save to captured_requests table
    const [captured] = await db
      .insert(capturedRequests)
      .values({
        endpointId: endpoint.id,
        requestId,
        httpMethod,
        headers,
        queryParams,
        rawBody,
        maskedBody: rawBody, // Initially same, worker will mask if PII detected
        parsedJson,
        clientIp,
        responseStatus,
        processingStatus: "RECEIVED",
      })
      .returning();

    // Asynchronously push to Valkey Stream for Security Pipeline workers (fire-and-forget, non-blocking)
    valkey
      .xadd(
        "stream:requests",
        "*",
        "requestId",
        captured.id,
        "endpointId",
        endpoint.id,
        "projectId",
        endpoint.projectId,
        "rawBody",
        rawBody || ""
      )
      .catch((err) => {
        logger.warn({ err: err.message, requestId }, "Failed to publish to Valkey stream (worker pipeline)");
      });

    return {
      status: responseStatus,
      body: responseBody,
      captured,
    };
  }
}
