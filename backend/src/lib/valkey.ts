import { Redis } from "ioredis";
import { config } from "./config.js";
import { logger } from "./logger.js";

export const valkey = new Redis(config.VALKEY_URL, {
  maxRetriesPerRequest: 3,
  retryStrategy(times) {
    const delay = Math.min(times * 100, 3000);
    return delay;
  },
  lazyConnect: true,
});

valkey.on("connect", () => {
  logger.info(" Connected to Valkey");
});

valkey.on("error", (err) => {
  logger.error({ err }, " Valkey connection error");
});
