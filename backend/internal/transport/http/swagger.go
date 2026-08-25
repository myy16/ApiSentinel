package http

import (
	_ "embed"
	"net/http"
)

const openAPIJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "ApiSentinel API & Webhook Security Gateway",
    "description": "High-performance Go-Centric API & Webhook Security Gateway with real-time inspection, PII/Secret detection, JSON Schema Contract enforcement, Mock Engine, Upstream Forwarding & DLQ recovery, and AI explanations.",
    "version": "1.0.0"
  },
  "servers": [
    {
      "url": "http://localhost:3001",
      "description": "Local Development Server"
    }
  ],
  "components": {
    "securitySchemes": {
      "BearerAuth": {
        "type": "http",
        "scheme": "bearer",
        "bearerFormat": "JWT"
      },
      "OrgHeader": {
        "type": "apiKey",
        "in": "header",
        "name": "x-organization-id"
      }
    }
  },
  "paths": {
    "/health": {
      "get": {
        "tags": ["System"],
        "summary": "Health Check",
        "description": "Returns backend service health status.",
        "responses": {
          "200": { "description": "Service is healthy" }
        }
      }
    },
    "/api/auth/register": {
      "post": {
        "tags": ["Auth"],
        "summary": "1. Kullanıcı Kaydı (Register)",
        "description": "Yeni kullanıcı hesabı ve varsayılan organizasyon oluşturur.",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["email", "password", "name", "organizationName"],
                "properties": {
                  "email": { "type": "string", "example": "developer@apisentinel.dev" },
                  "password": { "type": "string", "example": "SentinelPass123!" },
                  "name": { "type": "string", "example": "Ahmet Yılmaz" },
                  "organizationName": { "type": "string", "example": "Acme Dev Corp" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "Kullanıcı ve organizasyon oluşturuldu, JWT tokenlar döner" }
        }
      }
    },
    "/api/auth/login": {
      "post": {
        "tags": ["Auth"],
        "summary": "2. Kullanıcı Girişi (Login)",
        "description": "Mevcut kullanıcı girişi yapar ve accessToken ile organizasyon ID döner.",
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["email", "password"],
                "properties": {
                  "email": { "type": "string", "example": "developer@apisentinel.dev" },
                  "password": { "type": "string", "example": "SentinelPass123!" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Giriş başarılı, 24 saat geçerli accessToken döner" }
        }
      }
    },
    "/api/auth/me": {
      "get": {
        "tags": ["Auth"],
        "summary": "3. Aktif Kullanıcı & Organizasyon Bilgisi",
        "security": [{ "BearerAuth": [] }],
        "responses": {
          "200": { "description": "Kullanıcı detayları ve üye olunan organizasyonlar" }
        }
      }
    },
    "/api/projects": {
      "get": {
        "tags": ["Projects"],
        "summary": "4. Projeleri Listele",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "responses": {
          "200": { "description": "Organizasyona ait projeler listesi" }
        }
      },
      "post": {
        "tags": ["Projects"],
        "summary": "5. Yeni Proje Oluştur",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["name"],
                "properties": {
                  "name": { "type": "string", "example": "Ödeme Geçidi Projesi" },
                  "description": { "type": "string", "example": "Stripe ve iyzico webhookları için ana proje" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "Proje oluşturuldu" }
        }
      }
    },
    "/api/projects/{id}": {
      "get": {
        "tags": ["Projects"],
        "summary": "6. Proje Detayı",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Proje detayları" } }
      },
      "delete": {
        "tags": ["Projects"],
        "summary": "7. Proje Sil",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Proje başarıyla silindi" } }
      }
    },
    "/api/projects/{projectId}/endpoints": {
      "get": {
        "tags": ["Endpoints"],
        "summary": "8. Proje Endpoint'lerini Listele",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Endpoint listesi" } }
      },
      "post": {
        "tags": ["Endpoints"],
        "summary": "9. Yeni Webhook Endpoint'i Oluştur",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["name"],
                "properties": {
                  "name": { "type": "string", "example": "Stripe Ödemeler" },
                  "slug": { "type": "string", "example": "stripe-payments" },
                  "mode": { "type": "string", "enum": ["PASS", "BLOCK", "MOCK", "CAPTURE_ONLY"], "example": "PASS" },
                  "upstreamUrl": { "type": "string", "example": "https://api.mycompany.com/webhooks/stripe" }
                }
              }
            }
          }
        },
        "responses": { "201": { "description": "Endpoint oluşturuldu" } }
      }
    },
    "/api/projects/{projectId}/endpoints/{endpointId}": {
      "put": {
        "tags": ["Endpoints"],
        "summary": "10. Endpoint Düzenle",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [
          { "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "name": { "type": "string", "example": "Güncellenmiş Stripe Hook" },
                  "mode": { "type": "string", "enum": ["PASS", "BLOCK", "MOCK", "CAPTURE_ONLY"] },
                  "isActive": { "type": "boolean", "example": true },
                  "upstreamUrl": { "type": "string", "example": "https://postman-echo.com/post" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Endpoint güncellendi" } }
      },
      "delete": {
        "tags": ["Endpoints"],
        "summary": "11. Endpoint Sil",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [
          { "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } },
          { "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": { "200": { "description": "Endpoint silindi" } }
      }
    },
    "/hook/{slug}": {
      "post": {
        "tags": ["Ingestion Gateway"],
        "summary": "12. Canlı Webhook Ateşleme Testi",
        "description": "Oluşturulan slug adresine webhook gönderin. PII, Secret, JSON Schema ve Mock kontrolleri çalışır.",
        "parameters": [
          { "name": "slug", "in": "path", "required": true, "schema": { "type": "string", "example": "stripe-payments" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "example": {
                  "event": "payment.completed",
                  "order_id": "ORD-99881",
                  "amount": 4990,
                  "customer_email": "john.doe@example.com"
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "İstek yakalandı, temizlendi ve iletim kuyruğuna alındı" },
          "403": { "description": "Güvenlik politikası ihlali (BLOCK) veya JSON Schema uyuşmazlığı" }
        }
      }
    },
    "/api/endpoints/{endpointId}/schema": {
      "get": {
        "tags": ["JSON Schema Contracts"],
        "summary": "13. Endpoint JSON Schema Getir",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Aktif JSON Schema sözleşmesi" } }
      },
      "post": {
        "tags": ["JSON Schema Contracts"],
        "summary": "14. Endpoint JSON Schema Kaydet / Güncelle",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": { "type": "object" }
            }
          }
        },
        "responses": { "200": { "description": "JSON Schema başarıyla kaydedildi" } }
      },
      "delete": {
        "tags": ["JSON Schema Contracts"],
        "summary": "15. Endpoint JSON Schema Sil",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "JSON Schema sözleşmesi silindi" } }
      }
    },
    "/api/projects/{projectId}/findings": {
      "get": {
        "tags": ["Security Findings"],
        "summary": "16. Güvenlik İhlallerini Listele",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Veritabanında kaydedilen güvenlik bulguları listesi" } }
      }
    },
    "/api/projects/{projectId}/findings/stats": {
      "get": {
        "tags": ["Security Findings"],
        "summary": "17. Güvenlik İhlali İstatistikleri",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Kritik, yüksek ve toplam ihlal sayıları" } }
      }
    },
    "/api/endpoints/{endpointId}/forwarding": {
      "get": {
        "tags": ["Forwarding & DLQ"],
        "summary": "18. Forwarding Ayarlarını Getir",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Upstream iletim ayarları" } }
      },
      "post": {
        "tags": ["Forwarding & DLQ"],
        "summary": "19. Forwarding Ayarlarını Kaydet",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["targetUrl"],
                "properties": {
                  "targetUrl": { "type": "string", "example": "https://postman-echo.com/post" },
                  "maxRetries": { "type": "integer", "example": 3 },
                  "timeoutMs": { "type": "integer", "example": 5000 },
                  "isEnabled": { "type": "boolean", "example": true }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Forwarding ayarları kaydedildi" } }
      }
    },
    "/api/endpoints/{endpointId}/dlq": {
      "get": {
        "tags": ["Forwarding & DLQ"],
        "summary": "20. Dead Letter Queue Kayıtlarını Listele",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Başarısız iletim kuyruğu kayıtları" } }
      },
      "delete": {
        "tags": ["Forwarding & DLQ"],
        "summary": "21. Dead Letter Queue Temizle (Purge)",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "DLQ kayıtları silindi" } }
      }
    },
    "/api/dlq/{id}/retry": {
      "post": {
        "tags": ["Forwarding & DLQ"],
        "summary": "22. DLQ Kaydını Yeniden İlet (Retry)",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "İstek güncel upstream adresine başarıyla iletildi ve çözüldü" } }
      }
    },
    "/api/projects/{projectId}/requests": {
      "get": {
        "tags": ["Traffic & Requests"],
        "summary": "23. Yakalanan İstekleri Listele",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": { "200": { "description": "Trafik kayıtları" } }
      }
    },
    "/api/requests/{id}/replay": {
      "post": {
        "tags": ["Traffic & Requests"],
        "summary": "24. SSRF Korumalı Replay Ateşleme",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["targetUrl"],
                "properties": {
                  "targetUrl": { "type": "string", "example": "https://postman-echo.com/post" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "Replay başarıyla yürütüldü" } }
      }
    },
    "/api/ai/explain": {
      "post": {
        "tags": ["AI Explainer"],
        "summary": "25. Güvenli AI Kök Neden & Çözüm Danışmanı",
        "security": [{ "BearerAuth": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "properties": {
                  "category": { "type": "string", "example": "SECRET" },
                  "findingType": { "type": "string", "example": "AWS_KEY" },
                  "severity": { "type": "string", "example": "CRITICAL" },
                  "maskedEvidence": { "type": "string", "example": "AKIA****1234" },
                  "message": { "type": "string", "example": "AWS Access Key detected in payload" }
                }
              }
            }
          }
        },
        "responses": { "200": { "description": "AI güvenlik analizi ve çözüm önerisi" } }
      }
    }
  }
}`

const swaggerUIHTML = `<!DOCTYPE html>
<html lang="tr">
<head>
  <meta charset="UTF-8">
  <title>ApiSentinel — Swagger UI</title>
  <link rel="stylesheet" href="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui.css" />
  <link rel="icon" type="image/png" href="https://unpkg.com/swagger-ui-dist@5.11.0/favicon-32x32.png" />
  <style>
    body { margin: 0; background: #0b0f19; font-family: Inter, system-ui, sans-serif; }
    .swagger-ui .topbar { display: none; }
    .swagger-ui { color: #e2e8f0; }
    .swagger-ui .info .title { color: #6366f1; }
    .swagger-ui .scheme-container { background: #111827; border-bottom: 1px solid #1f2937; }
    .swagger-ui .opblock { border-radius: 8px; }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: "/swagger.json",
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [
          SwaggerUIBundle.presets.apis,
          SwaggerUIStandalonePreset
        ],
        layout: "BaseLayout"
      });
    };
  </script>
</body>
</html>`

func RegisterSwagger(r interface {
	Get(pattern string, h http.HandlerFunc)
}) {
	// 1. OpenAPI JSON Spec
	r.Get("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(openAPIJSON))
	})

	// 2. Swagger UI
	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(swaggerUIHTML))
	})

	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs", http.StatusMovedPermanently)
	})
}
