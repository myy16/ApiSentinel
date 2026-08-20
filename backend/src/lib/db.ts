import { drizzle } from "drizzle-orm/postgres-js";
import postgres from "postgres";
import { config } from "./config.js";
import * as schema from "../db/schema/index.js";

// For migrations and queries
const queryClient = postgres(config.DATABASE_URL, {
  max: 20,
  idle_timeout: 30,
  connect_timeout: 10,
});

export const db = drizzle(queryClient, { schema });
export type DB = typeof db;
