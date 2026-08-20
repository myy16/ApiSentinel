CREATE TABLE IF NOT EXISTS "captured_requests" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"endpoint_id" uuid NOT NULL,
	"request_id" varchar(100) NOT NULL,
	"http_method" varchar(10) NOT NULL,
	"headers" jsonb NOT NULL,
	"query_params" jsonb NOT NULL,
	"raw_body" text,
	"masked_body" text,
	"parsed_json" jsonb,
	"client_ip" "inet",
	"response_status" integer,
	"processing_status" varchar(30) DEFAULT 'RECEIVED',
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "captured_requests_request_id_unique" UNIQUE("request_id")
);
--> statement-breakpoint
CREATE TABLE IF NOT EXISTS "endpoints" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"project_id" uuid NOT NULL,
	"slug" varchar(128) NOT NULL,
	"name" varchar(100) NOT NULL,
	"mode" varchar(30) DEFAULT 'PASS' NOT NULL,
	"is_active" boolean DEFAULT true NOT NULL,
	"upstream_url" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	CONSTRAINT "endpoints_slug_unique" UNIQUE("slug")
);
--> statement-breakpoint
CREATE TABLE IF NOT EXISTS "mock_rules" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"endpoint_id" uuid NOT NULL,
	"name" varchar(150) NOT NULL,
	"condition" jsonb,
	"status_code" integer DEFAULT 200 NOT NULL,
	"delay_ms" integer DEFAULT 0 NOT NULL,
	"response_headers" jsonb,
	"response_body" jsonb,
	"enabled" boolean DEFAULT true NOT NULL
);
--> statement-breakpoint
CREATE TABLE IF NOT EXISTS "policies" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"project_id" uuid NOT NULL,
	"name" varchar(150) NOT NULL,
	"configuration" jsonb NOT NULL,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
CREATE TABLE IF NOT EXISTS "replay_jobs" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"source_request_id" uuid NOT NULL,
	"target_type" varchar(30) NOT NULL,
	"target_url" text,
	"status" varchar(30) DEFAULT 'PENDING' NOT NULL,
	"response_status" integer,
	"response_body" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL,
	"completed_at" timestamp with time zone
);
--> statement-breakpoint
CREATE TABLE IF NOT EXISTS "rules" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"name" varchar(150) NOT NULL,
	"category" varchar(50) NOT NULL,
	"rule_type" varchar(100) NOT NULL,
	"severity" varchar(20) DEFAULT 'HIGH' NOT NULL,
	"enabled" boolean DEFAULT true NOT NULL,
	"configuration" jsonb
);
--> statement-breakpoint
CREATE TABLE IF NOT EXISTS "security_findings" (
	"id" uuid PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
	"request_id" uuid NOT NULL,
	"rule_id" uuid,
	"category" varchar(50) NOT NULL,
	"type" varchar(100) NOT NULL,
	"severity" varchar(20) NOT NULL,
	"action" varchar(20) NOT NULL,
	"field_path" text,
	"message" text NOT NULL,
	"evidence_masked" text,
	"created_at" timestamp with time zone DEFAULT now() NOT NULL
);
--> statement-breakpoint
DO $$ BEGIN
 ALTER TABLE "captured_requests" ADD CONSTRAINT "captured_requests_endpoint_id_endpoints_id_fk" FOREIGN KEY ("endpoint_id") REFERENCES "public"."endpoints"("id") ON DELETE cascade ON UPDATE no action;
EXCEPTION
 WHEN duplicate_object THEN null;
END $$;
--> statement-breakpoint
DO $$ BEGIN
 ALTER TABLE "endpoints" ADD CONSTRAINT "endpoints_project_id_projects_id_fk" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE cascade ON UPDATE no action;
EXCEPTION
 WHEN duplicate_object THEN null;
END $$;
--> statement-breakpoint
DO $$ BEGIN
 ALTER TABLE "mock_rules" ADD CONSTRAINT "mock_rules_endpoint_id_endpoints_id_fk" FOREIGN KEY ("endpoint_id") REFERENCES "public"."endpoints"("id") ON DELETE cascade ON UPDATE no action;
EXCEPTION
 WHEN duplicate_object THEN null;
END $$;
--> statement-breakpoint
DO $$ BEGIN
 ALTER TABLE "policies" ADD CONSTRAINT "policies_project_id_projects_id_fk" FOREIGN KEY ("project_id") REFERENCES "public"."projects"("id") ON DELETE cascade ON UPDATE no action;
EXCEPTION
 WHEN duplicate_object THEN null;
END $$;
--> statement-breakpoint
DO $$ BEGIN
 ALTER TABLE "replay_jobs" ADD CONSTRAINT "replay_jobs_source_request_id_captured_requests_id_fk" FOREIGN KEY ("source_request_id") REFERENCES "public"."captured_requests"("id") ON DELETE cascade ON UPDATE no action;
EXCEPTION
 WHEN duplicate_object THEN null;
END $$;
--> statement-breakpoint
DO $$ BEGIN
 ALTER TABLE "security_findings" ADD CONSTRAINT "security_findings_request_id_captured_requests_id_fk" FOREIGN KEY ("request_id") REFERENCES "public"."captured_requests"("id") ON DELETE cascade ON UPDATE no action;
EXCEPTION
 WHEN duplicate_object THEN null;
END $$;
--> statement-breakpoint
DO $$ BEGIN
 ALTER TABLE "security_findings" ADD CONSTRAINT "security_findings_rule_id_rules_id_fk" FOREIGN KEY ("rule_id") REFERENCES "public"."rules"("id") ON DELETE set null ON UPDATE no action;
EXCEPTION
 WHEN duplicate_object THEN null;
END $$;
