import { defineConfig } from "drizzle-kit";

export default defineConfig({
  schema: "./src/db/schema/index.ts",
  out: "./drizzle",
  dialect: "postgresql",
  dbCredentials: {
    url: process.env.DATABASE_URL || "postgresql://apisentinel:apisentinel_secret@localhost:5432/apisentinel_db",
  },
  verbose: true,
  strict: true,
});
