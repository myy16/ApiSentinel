import { z } from "zod";
import dotenv from "dotenv";

dotenv.config({ path: "../.env" });

const configSchema = z.object({
  NODE_ENV: z.enum(["development", "production", "test"]).default("development"),
  PORT: z.coerce.number().default(3001),
  HOST: z.string().default("0.0.0.0"),
  DATABASE_URL: z.string().default("postgresql://apisentinel:apisentinel_secret@localhost:5432/apisentinel_db"),
  VALKEY_URL: z.string().default("redis://localhost:6379"),
  JWT_SECRET: z.string().min(32).default("super_secret_jwt_key_at_least_32_characters_long_12345"),
  JWT_EXPIRES_IN: z.string().default("15m"),
  REFRESH_TOKEN_EXPIRES_IN: z.string().default("7d"),
});

export const config = configSchema.parse(process.env);
