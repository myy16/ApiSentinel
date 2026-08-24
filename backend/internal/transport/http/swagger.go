package http

import (
	_ "embed"
	"net/http"
)

const openAPIJSON = `{
  "openapi": "3.0.3",
  "info": {
    "title": "ApiSentinel API & Webhook Security Gateway",
    "description": "High-performance Go-Centric API & Webhook Security Gateway with realtime inspection, PII/Secret detection, Replay Lab, and AI explanations.",
    "version": "0.1.0"
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
        "summary": "Health Check",
        "description": "Returns backend service health and version.",
        "responses": {
          "200": { "description": "Service is healthy" }
        }
      }
    },
    "/api/auth/register": {
      "post": {
        "summary": "1. Kullanıcı Kaydı (Register)",
        "description": "Yeni kullanıcı hesabı ve otomatik varsayılan organizasyon oluşturur.",
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
          "200": { "description": "Giriş başarılı, accessToken döner" }
        }
      }
    },
    "/api/projects": {
      "get": {
        "summary": "3. Projeleri Listele",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "responses": {
          "200": { "description": "Organizasyona ait projeler listesi" }
        }
      },
      "post": {
        "summary": "4. Yeni Proje Oluştur",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["name"],
                "properties": {
                  "name": { "type": "string", "example": "E-Commerce Gateway" },
                  "description": { "type": "string", "example": "Ödeme ve sipariş webhookları için ana proje" }
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
    "/api/projects/{projectId}/endpoints": {
      "get": {
        "summary": "5. Proje Endpoint'lerini Listele",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [
          { "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Endpoint listesi" }
        }
      },
      "post": {
        "summary": "6. Yeni Webhook Endpoint'i Oluştur",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [
          { "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["name"],
                "properties": {
                  "name": { "type": "string", "example": "Stripe Payments Hook" },
                  "description": { "type": "string", "example": "Stripe ödeme bildirimlerini dinler" },
                  "customSlug": { "type": "string", "example": "stripe-payments" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "Endpoint ve /hook/:slug URL'i oluşturuldu" }
        }
      }
    },
    "/hook/{slug}": {
      "post": {
        "summary": "7. Webhook Ateşleme Testi (Ingestion Gateway)",
        "description": "Oluşturulan slug adresine webhook gönderin. Güvenlik motoru anında PII, Secret ve Politika denetimi yapar.",
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
          "200": { "description": "İstek yakalandı, incelendi ve Valkey Stream'e aktarıldı" },
          "403": { "description": "Güvenlik politikası ihlali nedeniyle engellendi (BLOCK)" }
        }
      }
    },
    "/api/projects/{projectId}/requests": {
      "get": {
        "summary": "8. Yakalanan İstekleri Listele",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [
          { "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "responses": {
          "200": { "description": "Yakalanan istek kayıtları ve güvenlik bulguları" }
        }
      }
    },
    "/api/requests/{id}/replay": {
      "post": {
        "summary": "9. SSRF Korumalı Replay Ateşleme",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [
          { "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }
        ],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["targetUrl"],
                "properties": {
                  "targetUrl": { "type": "string", "example": "https://httpbin.org/post" }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Replay başarıyla yürütüldü ve gecikme ölçüldü" },
          "400": { "description": "SSRF Guard özel IP'yi engelledi" }
        }
      }
    },
    "/api/ai/explain": {
      "post": {
        "summary": "10. Güvenli AI Kök Neden & Çözüm Danışmanı",
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
        "responses": {
          "200": { "description": "AI Kök neden, etki analizi ve düzeltme kodu örneği döner" }
        }
      }
    },
    "/api/projects/{projectId}/alerts": {
      "post": {
        "summary": "11. Bildirim Kanalı Ekle (Slack / Discord / Telegram / Webhook)",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "requestBody": {
          "required": true,
          "content": {
            "application/json": {
              "schema": {
                "type": "object",
                "required": ["name", "channelType", "webhookUrl"],
                "properties": {
                  "name": { "type": "string", "example": "SecOps Slack" },
                  "channelType": { "type": "string", "enum": ["SLACK", "DISCORD", "TELEGRAM", "WEBHOOK"], "example": "SLACK" },
                  "webhookUrl": { "type": "string", "example": "https://hooks.slack.com/services/..." },
                  "minSeverity": { "type": "string", "example": "HIGH" }
                }
              }
            }
          }
        },
        "responses": {
          "201": { "description": "Bildirim kanalı başarıyla eklendi" }
        }
      },
      "get": {
        "summary": "12. Proje Bildirim Kanallarını Listele",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "projectId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": {
          "200": { "description": "Kayıtlı bildirim kanalları listesi" }
        }
      }
    },
    "/api/alerts/{id}/test": {
      "post": {
        "summary": "13. Test Bildirimi Gönder",
        "security": [{ "BearerAuth": [] }],
        "parameters": [{ "name": "id", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": {
          "200": { "description": "Test bildirimi hedefe başarıyla iletildi" }
        }
      }
    },
    "/api/endpoints/{endpointId}/forwarding": {
      "post": {
        "summary": "14. Upstream Forwarding Yapılandır",
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
                  "targetUrl": { "type": "string", "example": "https://api.mycompany.com/webhooks/stripe" },
                  "maxRetries": { "type": "integer", "example": 3 },
                  "timeoutMs": { "type": "integer", "example": 5000 },
                  "isEnabled": { "type": "boolean", "example": true }
                }
              }
            }
          }
        },
        "responses": {
          "200": { "description": "Upstream iletim ayarları güncellendi" }
        }
      },
      "get": {
        "summary": "15. Upstream Forwarding Ayarlarını Getir",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": {
          "200": { "description": "Aktif upstream forwarding konfigürasyonu" }
        }
      }
    },
    "/api/endpoints/{endpointId}/dlq": {
      "get": {
        "summary": "16. Dead Letter Queue (DLQ) Listele",
        "security": [{ "BearerAuth": [], "OrgHeader": [] }],
        "parameters": [{ "name": "endpointId", "in": "path", "required": true, "schema": { "type": "string" } }],
        "responses": {
          "200": { "description": "İletilemeyen başarısız webhook kayıtları" }
        }
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
    body { margin: 0; background: #0f172a; }
    .swagger-ui .topbar { display: none; }
    .swagger-ui { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; }
    .custom-header {
      background: #1e293b;
      color: #f8fafc;
      padding: 16px 24px;
      display: flex;
      align-items: center;
      justify-content: space-between;
      border-bottom: 1px solid #334155;
    }
    .custom-header h1 { margin: 0; font-size: 18px; display: flex; align-items: center; gap: 8px; }
    .badge {
      background: #3b82f6;
      color: white;
      font-size: 11px;
      padding: 3px 8px;
      border-radius: 9999px;
      font-weight: 600;
    }
  </style>
</head>
<body>
  <div class="custom-header">
    <h1>🛡️ ApiSentinel <span>Interactive API Explorer</span></h1>
    <span class="badge">Go-Centric v0.1.0</span>
  </div>
  <div id="swagger-ui"></div>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-bundle.js"></script>
  <script src="https://unpkg.com/swagger-ui-dist@5.11.0/swagger-ui-standalone-preset.js"></script>
  <script>
    window.onload = function() {
      window.ui = SwaggerUIBundle({
        url: "/openapi.json",
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
	Get(pattern string, handlerFn http.HandlerFunc)
}) {
	r.Get("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(openAPIJSON))
	})

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(swaggerUIHTML))
	})

	r.Get("/swagger", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/docs", http.StatusMovedPermanently)
	})
}
