import { z } from "zod";
import {
  UserRole,
  EndpointMode,
  Severity,
  PolicyAction,
  FindingCategory,
  ReplayTargetType,
} from "../constants/index";

// Auth Schemas
export const registerSchema = z.object({
  email: z.string().email("Geçerli bir e-posta adresi giriniz"),
  password: z.string().min(8, "Şifre en az 8 karakter olmalıdır"),
  organizationName: z.string().min(2, "Organizasyon adı en az 2 karakter olmalıdır").optional(),
});

export type RegisterInput = z.infer<typeof registerSchema>;

export const loginSchema = z.object({
  email: z.string().email("Geçerli bir e-posta adresi giriniz"),
  password: z.string().min(1, "Şifre gereklidir"),
});

export type LoginInput = z.infer<typeof loginSchema>;

export const refreshTokenSchema = z.object({
  refreshToken: z.string().min(1, "Refresh token gereklidir"),
});

export type RefreshTokenInput = z.infer<typeof refreshTokenSchema>;

// Project Schemas
export const createProjectSchema = z.object({
  name: z.string().min(2, "Proje adı en az 2 karakter olmalıdır").max(100),
});

export type CreateProjectInput = z.infer<typeof createProjectSchema>;

// Endpoint Schemas
export const createEndpointSchema = z.object({
  name: z.string().min(2, "Endpoint adı en az 2 karakter olmalıdır").max(100),
  slug: z
    .string()
    .min(3, "Slug en az 3 karakter olmalıdır")
    .max(100)
    .regex(/^[a-z0-9-]+$/, "Slug sadece küçük harf, rakam ve tire içerebilir")
    .optional(),
  mode: z.enum([EndpointMode.PASS, EndpointMode.BLOCK, EndpointMode.MOCK, EndpointMode.CAPTURE_ONLY]).default(EndpointMode.PASS),
  upstreamUrl: z.string().url("Geçerli bir URL giriniz").optional().nullable(),
});

export type CreateEndpointInput = z.infer<typeof createEndpointSchema>;

export const updateEndpointSchema = createEndpointSchema.partial().extend({
  isActive: z.boolean().optional(),
});

export type UpdateEndpointInput = z.infer<typeof updateEndpointSchema>;

// Policy Rule Schema
export const policyRuleSchema = z.object({
  category: z.nativeEnum(FindingCategory).optional(),
  type: z.string().optional(),
  severity: z.nativeEnum(Severity).optional(),
  action: z.nativeEnum(PolicyAction),
});

export const updatePolicySchema = z.object({
  name: z.string().min(2).max(100).optional(),
  rules: z.array(policyRuleSchema),
});

export type UpdatePolicyInput = z.infer<typeof updatePolicySchema>;

// Mock Rule Schema
export const createMockRuleSchema = z.object({
  name: z.string().min(2).max(100),
  condition: z.record(z.unknown()).optional().nullable(),
  statusCode: z.number().int().min(100).max(599).default(200),
  delayMs: z.number().int().min(0).max(30000).default(0),
  responseHeaders: z.record(z.string()).optional().nullable(),
  responseBody: z.unknown().optional().nullable(),
  enabled: z.boolean().default(true),
});

export type CreateMockRuleInput = z.infer<typeof createMockRuleSchema>;

// Replay Schema
export const createReplaySchema = z.object({
  targetType: z.enum([ReplayTargetType.PUBLIC_URL, ReplayTargetType.PROJECT_ENDPOINT, ReplayTargetType.LOCAL_AGENT]),
  targetUrl: z.string().url("Geçerli bir hedef URL giriniz").optional().nullable(),
});

export type CreateReplayInput = z.infer<typeof createReplaySchema>;
