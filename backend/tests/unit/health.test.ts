import { describe, it, expect } from "vitest";
import { buildApp } from "../../src/app.js";

describe("Healthcheck Endpoint", () => {
  it("should return 200 OK with service info", async () => {
    const app = buildApp();
    const response = await app.inject({
      method: "GET",
      url: "/health",
    });

    expect(response.statusCode).toBe(200);
    const body = JSON.parse(response.payload);
    expect(body.status).toBe("ok");
    expect(body.service).toBe("apisentinel-backend");
  });
});
