## Genel değerlendirme

Sistemin temel mimarisi çalışabilir durumda: PostgreSQL ve Valkey container’ları sağlıklı, frontend TypeScript kontrolü geçti. Ancak proje henüz production’a hazır değil; özellikle tenant isolation, hassas veri saklama ve webhook güvenliği tarafında yayın öncesi kapatılması gereken açıklar var.

## Kritik bulgular

1. Tenant isolation tam değil.

`RequireTenant` yalnızca kullanıcının gönderdiği organizasyona üye olduğunu doğruluyor. Fakat birçok işlemde istenen `projectId`, `endpointId`, `requestId` veya `dlqId` kaynağının bu organizasyona aitliği ayrıca doğrulanmıyor.

Örnekler:

- `POST /endpoints/{endpointId}/forwarding`
- `POST /endpoints/{endpointId}/schema`
- `POST /requests/{id}/replay`
- `DELETE /alerts/{id}`
- `POST /dlq/{id}/retry`

Bu endpoint’lerde yalnızca UUID bilen başka bir organizasyon üyesi, başka tenant’a ait kaynağa erişebilir/değiştirebilir. [router.go](D:/ApiSentinel/backend/internal/transport/http/router.go), [endpoint_service.go](D:/ApiSentinel/backend/internal/service/endpoint_service.go), [forwarding_handler.go](D:/ApiSentinel/backend/internal/transport/http/forwarding_handler.go)

Öneri: Her kaynak tabanlı route için `endpoint → project → organization` veya `request → endpoint → project → organization` zincirini SQL sorgusunda doğrula. `GetEndpointWithOwnership` sorgusu zaten var; bunu middleware veya servis katmanında standartlaştır.

2. Hassas veriler maskelenmiş olsa da ham haliyle saklanıyor ve API’den dönüyor.

Webhook body, headers ve query parametreleri doğrudan PostgreSQL’e yazılıyor. `masked_body` üretiliyor ama `raw_body` da korunuyor. Ayrıca request listesi hem `rawBody` hem `maskedBody` döndürüyor.

- [ingestion_service.go](D:/ApiSentinel/backend/internal/service/ingestion_service.go)
- [request_service.go](D:/ApiSentinel/backend/internal/service/request_service.go)
- [requests.sql](D:/ApiSentinel/backend/internal/database/queries/requests.sql)

Ek olarak Valkey stream’e de ham payload yazılıyor:

```go
"rawBody": string(rawBody)
```

Bu, PII, token, cookie, Authorization header veya ödeme verisinin PostgreSQL, Valkey ve dashboard’da kalmasına yol açabilir.

Öneri:

- Varsayılan olarak yalnızca maskelenmiş gövdeyi sakla ve göster.
- `Authorization`, `Cookie`, `Set-Cookie`, API key vb. header’ları yazmadan önce redact et.
- Ham payload gerekiyorsa ayrı, şifreli ve kısa retention’lı bir depoda tut.
- DLQ payload’ını da aynı kurala dahil et.
- Valkey stream için `MAXLEN`/retention tanımla; ham body yayınlama.

3. SSE şu an tenant guard sonrası çalışmayabilir.

Frontend SSE URL’sine sadece `token` ekleniyor:

```ts
?token=...
```

Ama backend tenant guard ayrıca organizasyon ID bekliyor. Frontend bu URL’ye `organizationId` eklemiyor. Sonuç olarak SSE büyük ihtimalle `TENANT_REQUIRED` ile kapanır.

- [useSSE.ts](D:/ApiSentinel/frontend/hooks/useSSE.ts)
- [auth.go](D:/ApiSentinel/backend/internal/middleware/auth.go)
- [router.go](D:/ApiSentinel/backend/internal/transport/http/router.go)

Ayrıca JWT’yi URL’ye koymak loglara, browser history’ye veya reverse proxy kayıtlarına sızma riski taşır.

Öneri: Kısa ömürlü, tek kullanımlık SSE ticket üret; ya da HttpOnly cookie ile SSE authentication kullan. Kısa vadede `organizationId` parametresini ekleyip backend’de doğrulat.

4. Public webhook endpoint’inde body limiti yok.

[ingestion_handler.go](D:/ApiSentinel/backend/internal/transport/http/ingestion_handler.go) şu anda:

```go
io.ReadAll(r.Body)
```

kullanıyor. Büyük bir request bellek tüketerek servis kesintisine yol açabilir. Ayrıca `X-Forwarded-For` doğrudan güvenilir kabul ediliyor; saldırgan rate-limit anahtarını sahte IP ile değiştirebilir.

Öneri:

- `http.MaxBytesReader` ile örneğin 1–5 MB limit koy.
- `X-Forwarded-For` yalnızca güvenilen reverse proxy arkasındayken kullan.
- Aksi halde `RemoteAddr` değerini `net.SplitHostPort` ile güvenli şekilde ayır.

5. HMAC doğrulaması global ve isteğe bağlı.

[ingestion_service.go](D:/ApiSentinel/backend/internal/service/ingestion_service.go) Stripe/GitHub/Shopify secret’larını tek global environment değişkeninden alıyor. Ayrıca signature header veya secret yoksa webhook çoğunlukla kabul ediliyor.

Bu multi-tenant yapı için yeterli değil: her endpoint’in kendi provider secret’ı olmalı.

Öneri: Endpoint bazlı, şifrelenmiş webhook secret alanı ekle. Endpoint ayarında “signature required” seçeneği bulunsun ve aktifse imzasız isteği kesin olarak reddet.

## Önemli bulgular

- Agent gRPC doğrulaması normal kullanıcı JWT’lerini de kabul ediyor; agent token’larında ayrı `audience`, `scope` veya `role` kontrolü yok. [server.go](D:/ApiSentinel/backend/internal/transport/grpc/server.go)

- Agent session listesi organizasyonla ilişkilendirilmemiş. Tenant guard geçse bile tüm bağlı ajanlar dönebilir. [agent_handler.go](D:/ApiSentinel/backend/internal/transport/http/agent_handler.go)

- Worker pool dolunca forwarding yeniden sınırsız goroutine’e düşüyor. Bu, worker pool’un koyduğu kaynak sınırını etkisiz kılar. [forwarding_service.go](D:/ApiSentinel/backend/internal/service/forwarding_service.go)

- Alert gönderimi de kontrolsüz `go func()` ile yapılıyor. Yoğun finding üretiminde goroutine artışı olabilir. [alert_service.go](D:/ApiSentinel/backend/internal/service/alert_service.go)

- Pagination limitlerinde üst sınır yok. Büyük `limit` değerleri çok fazla ham payload dönmesine ve DB yüküne neden olabilir. [request_handler.go](D:/ApiSentinel/backend/internal/transport/http/request_handler.go)

- `clientIP` ingestion fonksiyonuna iletiliyor ama `CreateCapturedRequest` çağrısında DB’ye yazılmıyor. Bu yüzden kayıtlar IP bilgisini kaybediyor. [ingestion_service.go](D:/ApiSentinel/backend/internal/service/ingestion_service.go)

- SSE handler CORS origin’i doğrudan `*` yapıyor; global CORS ayarını gölgeler. [sse_handler.go](D:/ApiSentinel/backend/internal/transport/http/sse_handler.go)

- SSRF doğrulaması DNS’i doğrulama anında çözüyor, HTTP client ise sonra tekrar çözebilir. DNS rebinding’e karşı custom `DialContext` ile hedef IP’yi bağlantı anında da doğrulamak daha sağlam olur. [ssrf.go](D:/ApiSentinel/backend/internal/security/ssrf/ssrf.go)

## Operasyon ve dokümantasyon eksikleri

- README eski Agent komutunu gösteriyor: `cd backend; go run ./cmd/agent ...`; bu komut artık mevcut değil. [README.md](D:/ApiSentinel/README.md)
- README “Valkey 8” diyor, compose dosyası `redis:7-alpine` kullanıyor. [docker-compose.yml](D:/ApiSentinel/docker-compose.yml)
- `Makefile clean` Windows ortamında çalışmaz; `rm -rf` kullanıyor.
- Migration ve `schema.sql` iki dosya kümesinde tutuluyor. Dokümantasyon bunu açıklasa da senkronizasyon manuel kalır; drift riski vardır.
- CI pipeline görünmüyor. En azından build, `go vet`, backend test, frontend typecheck, `sqlc generate` sonrası temiz diff ve secret scan çalışmalı.

## Önerdiğim uygulama sırası

1. Kaynak bazlı tenant ownership doğrulaması.
2. Raw payload/header/query redaction ve retention politikası.
3. Webhook body limiti, güvenilir proxy/IP kuralı, endpoint bazlı zorunlu HMAC.
4. SSE’nin organization bilgisini taşıması ve URL token yerine güvenli ticket/cookie modeli.
5. Agent token scope’ları ve organization-scoped agent sessions.
6. Worker pool fallback goroutine’lerini kaldırıp queue/backpressure politikası eklemek.
7. Pagination limitleri, replay response/body limitleri, Valkey stream retention.
8. CI ve README/Makefile düzeltmeleri.

Not: Audit sırasında frontend typecheck, takip edilen `frontend/tsconfig.tsbuildinfo` dosyasını değiştirdi; kaynak kod değişikliği değil, derleme metadatası. Git sandbox izni nedeniyle otomatik geri alamadım.