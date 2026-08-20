import { buildApp } from "./app.js";
import { config } from "./lib/config.js";
import { logger } from "./lib/logger.js";
import { valkey } from "./lib/valkey.js";

async function main() {
  const app = buildApp();

  try {
    // Attempt Valkey connection (lazyConnect)
    valkey.connect().catch((err) => {
      logger.warn({ err: err.message }, "Valkey is currently unreachable (make sure docker compose is running)");
    });

    await app.listen({ port: config.PORT, host: config.HOST });
    logger.info(`🚀 ApiSentinel Backend running on http://${config.HOST}:${config.PORT}`);
  } catch (err) {
    logger.error({ err }, "Failed to start server");
    process.exit(1);
  }
}

main();
