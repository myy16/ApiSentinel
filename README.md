# ApiSentinel

ApiSentinel, webhook ve API trafiğini yakalayan, derinlemesine güvenlik taramaları (Secret, PII, SQLi/XSS Injection, SSRF koruması, Rate Limiting, HMAC doğrulama) gerçekleştiren, mock/replay özellikleri sunan ve Go tabanlı Shift-Left Local CLI Ajanı ile commit öncesi güvenlik denetimi sağlayan developer-first API Security & Observability platformudur.

## Mimari & Teknoloji Yığını

- **Backend:** Go 1.25+ (Chi Router, gRPC bi-directional streaming, sqlc, PostgreSQL, Valkey / Redis)
- **Frontend:** Next.js 14 (App Router, TailwindCSS, TanStack React Query, Server-Sent Events - SSE)
- **Local Agent / CLI:** Go (Git Pre-Commit / Pre-Push Hook, Shannon Entropy & Regex Secret Scanner)
- **Protokol:** HTTP/REST, SSE (Realtime Dashboard), gRPC / Protobuf (Agent Tunnel)

## Gereksinimler

- Go 1.25+
- Node.js 20+
- Docker & Docker Compose (PostgreSQL 16, Redis 7-compatible Valkey service)

## Yerel Geliştirme

1. **Ortam değişkenlerini hazırlayın:**
   ```powershell
   Copy-Item .env.example .env
   ```
2. **Altyapıyı başlatın:**
   ```powershell
   docker compose up -d
   ```
3. **Backend'i başlatın:**
   ```powershell
   cd backend
   go run ./cmd/server
   ```
4. **Frontend'i başlatın:**
   ```powershell
   cd frontend
   npm run dev
   ```
5. **Local Ajanı (CLI) Çalıştırın / Bağlayın:**
   ```powershell
   cd agent
   go run ./cmd/apisentinel connect --server localhost:50051 --token <API_KEY>
   ```

- **Backend API Gateway & Docs:** `http://localhost:3001` (Swagger: `/swagger` veya `/docs`)
- **Frontend Dashboard:** `http://localhost:3000`
- **gRPC Agent Port:** `localhost:50051`

### Endpoint webhook imza doğrulaması

Her endpoint için imza ayarı, API üzerinden saklanır; secret yalnızca şifrelenmiş biçimde veritabanına yazılır ve bir daha okunarak dönülmez. Önce `WEBHOOK_SECRET_ENCRYPTION_KEY` ayarlanmalıdır. Swagger üzerinden şu çağrıyı yapabilirsin:

```text
PUT /api/endpoints/{endpointId}/webhook-security
```

```json
{
  "provider": "stripe",
  "secret": "whsec_...",
  "requireSignature": true
}
```

Agent bağlantısı için dashboard JWT'si yerine o projeye ait bir API key kullanılır. Production'da TLS açıksa Agent'a `APISENTINEL_GRPC_TLS=true` ve gerektiğinde `APISENTINEL_GRPC_SERVER_NAME` verilir.

---

## Veritabanı Şeması & Migration Yönetimi (Tek Doğruluk Kaynağı)

Veritabanı şemasının tek doğruluk kaynağı (Single Source of Truth) `backend/internal/database/` dizinidir:
- **Migration Dosyaları:** `backend/internal/database/migrations/*.sql` (Uygulama açılışında `migrator.go` tarafından otomatik olarak sırayla uygulanır).
- **sqlc Şema Tanımı:** `backend/internal/database/schema.sql` (Migration'ların birleşimidir; sqlc kod üretimi için kullanılır).
- **Tip Güvenli Sorgular:** `backend/internal/database/queries/*.sql` -> `sqlc generate` ile tip güvenli Go koduna dönüştürülür.

---

## Test ve Doğrulama

```powershell
# Backend unit & integration testleri
cd backend
go test -v ./internal/...

# Security Engine Benchmark
go test -bench=. ./internal/security

# Statik Analiz (0 warning)
go vet ./...
```
