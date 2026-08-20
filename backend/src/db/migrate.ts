import { migrate } from "drizzle-orm/postgres-js/migrator";
import { db } from "../lib/db.js";
import postgres from "postgres";
import { config } from "../lib/config.js";
import { logger } from "../lib/logger.js";
import { drizzle } from "drizzle-orm/postgres-js";

async function runMigrations() {
  logger.info("⏳ Running database migrations...");

  const migrationClient = postgres(config.DATABASE_URL, { max: 1 });
  const migrationDb = drizzle(migrationClient);

  try {
    await migrate(migrationDb, { migrationsFolder: "./drizzle" });
    logger.info("✅ Database migrations completed successfully!");
  } catch (err) {
    logger.error({ err }, "❌ Database migration failed");
    process.exit(1);
  } finally {
    await migrationClient.end();
  }
}

runMigrations();
