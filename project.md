from pathlib import Path


content = r"""# ApiSentinel — Uçtan Uca Proje Spesifikasyonu


## 0. Belgenin Amacı


Bu doküman, ApiSentinel projesini geliştirecek bir AI coding agent'ın projeyi sıfırdan anlayabilmesi, mimari kararları koruyabilmesi, MVP'yi çalışır hâle getirebilmesi ve daha sonra güvenli şekilde genişletebilmesi için hazırlanmıştır.


Bu belge yalnızca fikir açıklaması değildir. Aynı zamanda:
- ürün gereksinimi,
- teknik mimari,
- güvenlik modeli,
- veri modeli,
- API sözleşmeleri,
- Local Agent davranışı,
- Git entegrasyonu,
- API/Webhook Gateway davranışı,
- Rule Engine,
- Replay/Mock/Chaos sistemi,
- frontend bilgi mimarisi,
- geliştirme sırası,
- test stratejisi,
- deployment modeli,
- güvenlik ve KVKK yaklaşımı
için referans kaynağıdır.


AI agent bu dosyayı okuduktan sonra projeyi "basit webhook listener" olarak yorumlamamalıdır.


---


# 1. Ürün Tanımı


## 1.1 Proje Adı


**ApiSentinel**


Alternatif isimler:
- HookGuard
- DevSentinel
- Integration Sentinel


Şimdilik canonical ürün adı: **ApiSentinel**


## 1.2 Ürün Tanımı


ApiSentinel; geliştiricilerin ve ekiplerin yazılım geliştirme sürecinde Git repository'leri, local API'ler ve dış servislerden gelen API/Webhook trafiği üzerinde güvenlik, hassas veri, entegrasyon güvenilirliği ve veri sözleşmesi kontrolleri yapmasını sağlayan bir **Developer Security & Integration Platform**'dur.


Temel yaklaşım:


```text
DEVELOP
   ↓
DETECT
   ↓
DECIDE
   ↓
PREVENT / BLOCK / ALERT
   ↓
TEST
   ↓
OBSERVE
1.3 Temel Problem

Modern uygulamalar çok sayıda dış servis ve mikroservis ile veri alışverişi yapar.

Örnek:

GitHub
Stripe
iyzico
Bank API
CRM
Cargo API
Payment Service
Order Service
Notification Service

Bu entegrasyonlarda şu problemler ortaya çıkabilir:

API key / secret'ın Git repository'ye gönderilmesi
Hassas kişisel verilerin API payload'ında taşınması
Authorization token veya secret'ın loglanması
Şüpheli SQL Injection / XSS payload'ları
Webhook'un birden fazla kez gönderilmesi
Webhook schema'sının bozulması
Dış servisin 500/503 dönmesi
Dış servisin yavaş cevap vermesi
Retry mekanizmasının düzgün çalışmaması
Malformed JSON
Beklenmeyen alanlar
Eksik zorunlu alanlar
Entegrasyon hatalarının production öncesinde test edilememesi

ApiSentinel bu problemlerin bir kısmını kaynakta, bir kısmını gateway seviyesinde, bir kısmını ise gözlemleme/analiz seviyesinde ele alır.

2. Ürün Felsefesi

ApiSentinel'in temel prensipleri:

2.1 Prevention First

Risk mümkünse veri sistemden çıkmadan veya backend'e ulaşmadan engellenmelidir.

2.2 Detection When Prevention Is Impossible

Bazı runtime olayları ancak gerçek API trafiği görüldüğünde tespit edilebilir.

2.3 Policy Driven

Her bulgu otomatik olarak BLOCK edilmemelidir.

Kullanıcı veya organizasyon policy belirleyebilmelidir.

Örnek:

API_KEY       -> BLOCK
CREDIT_CARD   -> BLOCK
TCKN          -> BLOCK
EMAIL         -> MASK
SQLI          -> ALERT
DUPLICATE     -> WARN
SCHEMA_ERROR  -> BLOCK
2.4 Deterministic Security

Security detection'ın temelini deterministik rule engine oluşturur.

LLM güvenlik kararının tek kaynağı değildir.

2.5 AI as Explanation, Not Authority

AI:

bulguyu açıklar,
remediation önerir,
test case üretir,
olayları özetler.

AI tek başına:

"bu kesin güvenli",
"bu kesin saldırı"
gibi kritik kararların otoritesi değildir.
2.6 Privacy by Design

Hassas veri tespit edildiğinde mümkün olduğunca:

maskelenmeli,
minimize edilmeli,
retention uygulanmalı,
erişim sınırlandırılmalı,
gereksiz yere üçüncü taraf AI servisine gönderilmemelidir.
3. Ürün Kapsamı

ApiSentinel dört ana bileşenden oluşur.

                    ApiSentinel
                         |
       +-----------------+------------------+
       |                 |                  |
       v                 v                  v
 Local Agent       Gateway/API         Control Plane
       |                 |                  |
       v                 v                  v
 Git Security       Runtime Security     Dashboard
 Local API          Webhook Analysis     Policy
 Replay             Mock/Chaos           Analytics
3.1 Local Agent

Geliştiricinin bilgisayarında çalışır.

Görevleri:

Git hook entegrasyonu
Secret scanning
local API replay
cloud ile güvenli bağlantı
local target'a istek yönlendirme
gerektiğinde local event capture
3.2 Ingestion / Gateway

Dış API/Webhook isteklerini karşılar.

Görevleri:

endpoint identification
authentication/signature validation
rate limiting
request normalization
request ID üretimi
policy değerlendirmesi
security analysis
mock response
block/allow kararı
event queue'ya gönderme
3.3 Analysis Pipeline

Asenkron analiz katmanıdır.

Görevleri:

PII detection
secret detection
injection detection
schema validation
duplicate detection
security finding oluşturma
severity belirleme
event persistence
3.4 Control Plane / Dashboard

Next.js tabanlı kullanıcı arayüzüdür.

Görevleri:

project yönetimi
endpoint yönetimi
canlı event akışı
request inspector
security findings
replay
mock rules
policies
contracts
analytics
agent yönetimi
4. Önerilen Teknoloji Stack
Backend

Spring Boot + Java 17 veya üzeri

Neden:

güçlü REST altyapısı
Spring Security
WebSocket/SSE
PostgreSQL entegrasyonu
kurumsal mimariye uygunluk
test ekosistemi
mevcut ekip yetkinliği
Frontend

Next.js + TypeScript

Önerilen:

App Router
Tailwind CSS
shadcn/ui
TanStack Query
Zod
Monaco Editor veya uygun JSON viewer
Database

PostgreSQL

Queue / Event Streaming

MVP:
Redis Streams

Daha ileri aşama:
Kafka

Kafka MVP'ye başlangıçta eklenmemelidir.

Cache / Rate Limit

Redis

Local Agent

İlk MVP için:

Go veya Node.js

Öneri:
Go

Neden:

tek binary
düşük runtime dependency
CLI için uygun
cross-platform deployment kolaylığı
local network tooling için uygun
Authentication

MVP:

JWT + refresh token

İleri aşama:

OAuth2/OIDC
GitHub login
Google login
SSO/SAML
5. Yüksek Seviyeli Mimari
                         INTERNET
                            |
             +--------------+--------------+
             |                             |
       Git Provider                  API/Webhook
             |                             |
             v                             v
      Git Integration              Ingestion Gateway
             |                             |
             v                             v
       Cloud Analysis               Policy Engine
                                           |
                                           v
                                  Security Analysis
                                           |
                                           v
                                      Redis Streams
                                           |
                     +---------------------+---------------------+
                     |                     |                     |
                     v                     v                     v
                Security Worker       Schema Worker       Duplicate Worker
                     |                     |                     |
                     +---------------------+---------------------+
                                           |
                                           v
                                      PostgreSQL
                                           |
                         +-----------------+----------------+
                         |                                  |
                         v                                  v
                    SSE/WebSocket                     Replay Engine
                         |                                  |
                         v                                  v
                    Next.js UI                    Local Agent / Target API
6. Kritik Mimari Prensip: Synchronous vs Asynchronous
6.1 Synchronous Path

Gateway'in kullanıcıya/backend'e cevap vermesi gereken kritik yol:

Request
  ↓
Endpoint Lookup
  ↓
Authentication / Signature
  ↓
Rate Limit
  ↓
Policy Decision
  ↓
Optional Fast Security Checks
  ↓
Allow / Block / Mock
  ↓
HTTP Response

Bu yol çok hızlı olmalıdır.

6.2 Asynchronous Path

Daha ağır analizler:

Request
  ↓
Event
  ↓
Redis Stream
  ↓
Worker
  ↓
Analysis
  ↓
PostgreSQL
  ↓
Realtime Notification

Ağır analizler webhook response latency'yi gereksiz yere artırmamalıdır.

7. Request Lifecycle

Bir webhook geldiğinde:

1. HTTP request received
2. request_id generated
3. endpoint resolved
4. endpoint active check
5. rate limit check
6. authentication/signature check
7. mock rule check
8. synchronous policy check
9. event accepted or blocked
10. event normalized
11. event published to Redis Stream
12. worker analysis
13. findings created
14. event persisted
15. realtime update emitted
16. dashboard updated

Örnek:

POST /hook/payments-prod


Headers:
X-Signature: ...
Content-Type: application/json


Body:
{
  "event": "payment.success",
  "amount": 1000,
  "customer": {
    "email": "test@example.com"
  }
}

Sistem:

request ID üretir
endpoint'i bulur
signature kontrol eder
policy değerlendirir
event'i kaydeder
PII scanner çalıştırır
dashboard'a gönderir.
8. Local Agent Mimarisi
8.1 Temel Amaç

Local Agent iki önemli problemi çözer:

Git push/commit öncesi secret detection
Cloud'dan local API'lere güvenli replay
8.2 Agent Çalışma Modeli
Developer Machine


+----------------------------------+
| ApiSentinel Agent                |
|                                  |
| Git Hook Manager                 |
| Secret Scanner                   |
| Local HTTP Proxy (optional)      |
| Replay Receiver                  |
| Secure Tunnel                    |
+----------------+-----------------+
                 |
                 | outbound TLS
                 v
         ApiSentinel Cloud

Agent internete inbound port açmamalıdır.

Bağlantıyı Agent kendisi başlatmalıdır.

8.3 CLI

Örnek komutlar:

apisentinel login
apisentinel init
apisentinel connect
apisentinel status
apisentinel scan
apisentinel scan --staged
apisentinel install-hook
apisentinel uninstall-hook
apisentinel replay <event-id>
apisentinel config
8.4 Agent Authentication

Agent için kısa ömürlü veya rotate edilebilir token kullanılmalıdır.

Token:

plaintext loglanmamalı
config dosyasında açık tutulmamalı
mümkünse OS keychain kullanmalı
revoke edilebilir olmalı
9. Git Security Akışı
9.1 Amaç

Secret'ın Git remote'a gönderilmeden önce tespit edilmesi.

git push
   ↓
pre-push hook
   ↓
ApiSentinel Agent
   ↓
Local Secret Scanner
   ↓
Findings
   ↓
Policy
   |
   +--> BLOCK
   |
   +--> WARN
   |
   +--> ALLOW
9.2 Neden pre-push?

Pre-commit de kullanılabilir ancak:

geliştirici deneyimini gereksiz bozabilir
her commit'te tarama maliyeti oluşturabilir

MVP:
pre-push

İleri aşama:

pre-commit optional
IDE extension optional
CI scan
9.3 Secret Detection

Başlangıçta:

API key pattern'leri
AWS-like key pattern'leri
GitHub token pattern'leri
JWT
private key block
database URL/password
Stripe-like secret key pattern'leri
generic high-entropy secret detection

kullanılabilir.

Secret detection mümkün olduğunca false positive azaltacak şekilde tasarlanmalıdır.

9.4 Sonuç

Örnek:

CRITICAL: Secret detected
File: application.properties
Line: 14
Type: API_KEY
Action: BLOCK

Exit code non-zero olmalıdır.

10. Cloud Git Integration

Local hook'tan kaçış ihtimaline karşı ikinci katmandır.

Örneğin:

GitHub webhook
GitLab webhook
CI integration

ile commit/PR taraması yapılabilir.

Bu katman:

local scan yerine geçmez
defense-in-depth sağlar
11. API/Webhook Gateway
11.1 Endpoint

Örnek:

POST /hook/{slug}

veya:

ANY /hook/{slug}
11.2 Endpoint Slug

Tahmin edilmesi zor olmalıdır.

Örnek:

/hook/9a2c7d8f-pay-prod

Endpoint ID ile slug aynı şey olmamalıdır.

11.3 Gateway Response Modes

Endpoint şu modlara sahip olabilir:

PASS

Gerçek backend'e ilet.

BLOCK

İsteği reddet.

MOCK

Tanımlı mock response döndür.

CAPTURE_ONLY

İsteği kaydet, backend'e iletme.

12. Webhook Signature Verification

Platform mümkün olduğunca sağlayıcıya özel signature verification desteklemelidir.

Örnek:

HMAC SHA-256
timestamp validation
replay protection

İlk MVP'de generic HMAC:

HMAC(secret, timestamp + "." + rawBody)

gibi bir yapı kullanılabilir.

Daha sonra:

Stripe
GitHub
iyzico
özel HMAC provider
eklenebilir.

Secret'lar plaintext dashboard'da gösterilmemelidir.

13. Rate Limiting

Token Bucket veya Sliding Window kullanılabilir.

Örnek policy:

100 requests / minute / endpoint

Aşılırsa:

429 Too Many Requests

dönülür.

Rate limit key:

endpoint
API key
IP
tenant
gibi seçeneklere göre yapılandırılabilir.
14. Security Rule Engine

Rule Engine merkezi bir abstraction olmalıdır.

Önerilen model:

Rule
 ├── id
 ├── name
 ├── category
 ├── severity
 ├── detector
 ├── action
 ├── enabled
 └── configuration

Kategoriler:

PII
SECRET
INJECTION
AUTH
SCHEMA
DUPLICATE
DATA_QUALITY
15. PII Detection

İlk destek:

email
phone
TCKN
credit card candidate
IBAN candidate
address-like data (ileride)
15.1 Email

Regex + validation.

15.2 Phone

Türkiye ve uluslararası formatlar desteklenebilir.

15.3 TCKN

Sadece regex yeterli değildir.

11 hane kontrolü + T.C. kimlik numarası doğrulama algoritması uygulanmalıdır.

15.4 Credit Card

Regex + Luhn doğrulaması birlikte kullanılmalıdır.

Sadece regex eşleşmesi "kesin kredi kartı" kabul edilmemelidir.

15.5 Masking

Örnek:

4111111111111111

yerine:

************1111

gösterilir.

Email:

a***@example.com
16. Secret Detection Runtime

API payload'larında da secret aranabilir.

Örnek:

{
  "apiKey": "sk_live_xxx",
  "accessToken": "eyJ..."
}

Finding:

CRITICAL
Secret Exposure
field: apiKey

Raw secret dashboard'da gösterilmemelidir.

17. Injection Detection

İlk aşamada heuristic rule engine.

Destek:

SQL injection indicators
XSS indicators
NoSQL injection indicators

Önemli:

Bu scanner'ın çıktısı:

POTENTIAL_SQL_INJECTION

olmalıdır.

"Kesin saldırı" iddiası yapılmamalıdır.

False positive yönetimi bulunmalıdır.

18. Idempotency / Duplicate Detection

Amaç aynı event'in tekrar geldiğini belirlemek.

Hash:

SHA-256(normalized request)

Örnek pencere:

5 seconds

Aynı:

endpoint
method
normalized body
relevant headers

kombinasyonu tekrar gelirse duplicate adayı oluşturulur.

Ancak gerçek idempotency için mümkünse provider event ID desteklenmelidir.

Örnek:

X-Event-ID: evt_123

Aynı event ID tekrar geldiyse güçlü duplicate sinyali kabul edilir.

19. Schema / Contract Validation

Kullanıcı endpoint için JSON Schema tanımlayabilmelidir.

Örnek:

{
  "type": "object",
  "required": ["event", "amount"],
  "properties": {
    "event": {
      "type": "string"
    },
    "amount": {
      "type": "number"
    }
  }
}

Gelen payload:

{
  "event": "payment.success",
  "amount": "100"
}

Finding:

SCHEMA_VIOLATION
amount expected number, received string
20. Policy Engine

Policy engine bütün bulguların aksiyonunu belirler.

Örnek:

rules:
  - category: SECRET
    severity: CRITICAL
    action: BLOCK


  - category: PII
    type: CREDIT_CARD
    action: BLOCK


  - category: PII
    type: EMAIL
    action: MASK


  - category: SQL_INJECTION
    action: ALERT


  - category: DUPLICATE
    action: WARN

Action enum:

ALLOW
WARN
ALERT
MASK
BLOCK

Policy tenant/project/endpoint seviyelerinde override edilebilir.

21. Replay Engine

Her captured request replay edilebilir.

Replay hedefleri:

LOCAL_AGENT
PUBLIC_URL
PROJECT_ENDPOINT

Replay request:

POST /api/replay/{requestId}

ReplayJob oluşturulur.

ReplayJob
 ├── source_request_id
 ├── target_type
 ├── target_url
 ├── status
 ├── started_at
 ├── completed_at
 ├── response_status
 ├── response_headers
 └── response_body

Replay sırasında secret/PII policy tekrar uygulanmalıdır.

22. Local Replay

Cloud server doğrudan localhost'a erişmez.

Doğru akış:

Dashboard
   ↓
Replay API
   ↓
Cloud
   ↓
Agent WebSocket/TLS channel
   ↓
Agent
   ↓
localhost:8080

Agent bağlantıyı outbound olarak açar.

23. Mock Engine

Endpoint için response tanımlanabilir.

Alanlar:

status_code
delay_ms
headers
body
condition

Örnek:

{
  "statusCode": 503,
  "delayMs": 3000,
  "headers": {
    "Content-Type": "application/json"
  },
  "body": {
    "error": "service_unavailable"
  }
}

Koşullu mock:

WHEN body.event == "payment.failed"
THEN 500
24. Chaos / Failure Testing

İleri MVP özelliği.

Senaryolar:

latency injection
HTTP 500
HTTP 503
timeout simulation
malformed JSON
duplicate event
missing field
invalid signature
large payload
rate-limit response

Amaç:

Entegrasyonun hata durumlarında nasıl davrandığını production'ı bozmadan test etmek.

25. AI Layer

AI yalnızca güvenli şekilde hazırlanmış veriler üzerinde çalışmalıdır.

Pipeline:

Raw Event
   ↓
Security Scanner
   ↓
Mask Sensitive Data
   ↓
Safe Representation
   ↓
LLM

AI görevleri:

Finding Explanation

"Bu bulgu ne anlama geliyor?"

Remediation

"Nasıl düzeltilir?"

Test Generation

"Bu bulguyu doğrulamak için hangi test yapılabilir?"

Event Summary

"Son 1 saatte hangi entegrasyon problemleri yaşandı?"

Incident Summary

"500 hatalarının ortak nedeni ne olabilir?"

AI kararlarının yanında deterministic finding ID ve rule bilgisi tutulmalıdır.

26. Data Model

Ana entity'ler:

User
Organization
Project
Membership
Endpoint
CapturedRequest
SecurityFinding
Rule
Policy
MockRule
Contract
ReplayJob
Agent
ApiCredential
AuditLog
26.1 users
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
26.2 organizations
CREATE TABLE organizations (
    id UUID PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
26.3 memberships
CREATE TABLE memberships (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID NOT NULL REFERENCES users(id),
    role VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, user_id)
);

Roles:

OWNER
ADMIN
DEVELOPER
VIEWER
26.4 projects
CREATE TABLE projects (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(150) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
26.5 endpoints
CREATE TABLE endpoints (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id),
    slug VARCHAR(128) UNIQUE NOT NULL,
    name VARCHAR(100) NOT NULL,
    mode VARCHAR(30) NOT NULL DEFAULT 'PASS',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    upstream_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
26.6 captured_requests
CREATE TABLE captured_requests (
    id UUID PRIMARY KEY,
    endpoint_id UUID NOT NULL REFERENCES endpoints(id),
    request_id VARCHAR(100) UNIQUE NOT NULL,
    http_method VARCHAR(10) NOT NULL,
    headers JSONB,
    query_params JSONB,
    raw_body TEXT,
    masked_body TEXT,
    parsed_json JSONB,
    client_ip INET,
    response_status INT,
    processing_status VARCHAR(30),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

Not:

raw_body forever
unencrypted sensitive data forever

Retention policy uygulanmalıdır.

26.7 security_findings
CREATE TABLE security_findings (
    id UUID PRIMARY KEY,
    request_id UUID NOT NULL REFERENCES captured_requests(id),
    rule_id UUID,
    category VARCHAR(50) NOT NULL,
    type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    action VARCHAR(20) NOT NULL,
    field_path TEXT,
    message TEXT NOT NULL,
    evidence_masked TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
26.8 rules
CREATE TABLE rules (
    id UUID PRIMARY KEY,
    name VARCHAR(150) NOT NULL,
    category VARCHAR(50) NOT NULL,
    rule_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    configuration JSONB
);
26.9 policies
CREATE TABLE policies (
    id UUID PRIMARY KEY,
    project_id UUID NOT NULL REFERENCES projects(id),
    name VARCHAR(150) NOT NULL,
    configuration JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
26.10 mock_rules
CREATE TABLE mock_rules (
    id UUID PRIMARY KEY,
    endpoint_id UUID NOT NULL REFERENCES endpoints(id),
    name VARCHAR(150) NOT NULL,
    condition JSONB,
    status_code INT NOT NULL,
    delay_ms INT NOT NULL DEFAULT 0,
    response_headers JSONB,
    response_body JSONB,
    enabled BOOLEAN NOT NULL DEFAULT TRUE
);
26.11 replay_jobs
CREATE TABLE replay_jobs (
    id UUID PRIMARY KEY,
    source_request_id UUID NOT NULL REFERENCES captured_requests(id),
    target_type VARCHAR(30) NOT NULL,
    target_url TEXT,
    status VARCHAR(30) NOT NULL,
    response_status INT,
    response_body TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);
26.12 agents
CREATE TABLE agents (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    name VARCHAR(150) NOT NULL,
    status VARCHAR(30) NOT NULL,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
27. Multi-Tenancy

Organization bazlı tenant isolation zorunludur.

Her sorgu mümkün olduğunca:

organization_id

bağlamı ile çalışmalıdır.

Bir tenant'ın:

project
endpoint
request
finding
replay
agent
verileri başka tenant tarafından erişilememelidir.

Authorization sadece frontend'de yapılmamalıdır.

Backend'de zorunludur.

28. Hassas Veri Saklama

Raw payload güvenlik açısından hassastır.

Önerilen:

Raw payload
    ↓
Encryption at rest
    ↓
Strict access control
    ↓
Retention
    ↓
Deletion

Dashboard default olarak masked payload göstermelidir.

Örneğin:

{
  "email": "a***@example.com",
  "card": "************1111"
}

Raw veriyi görüntüleme:

özel permission
audit log
gerekirse re-authentication

gerektirebilir.

29. KVKK Yaklaşımı

ApiSentinel "KVKK uyumluluk sertifikası" veya "KVKK'yı garanti eder" şeklinde konumlandırılmamalıdır.

Doğru ifade:

"KVKK kapsamında korunması gereken kişisel verilerin tespit edilmesine ve güvenli şekilde yönetilmesine yardımcı olur."

Özellikler:

data minimization
masking
retention
deletion
access control
audit log
tenant isolation
encryption
30. BDDK / Regülasyon Konumlandırması

ApiSentinel BDDK tarafından onaylanmış veya BDDK uyumluluğunu garanti eden bir sistem olarak tanımlanmamalıdır.

Finans sektöründeki şirketler için:

sensitive data detection
auditability
policy enforcement
access control
security monitoring

gibi teknik kontroller sağlayabilir.

Sektörel uyumluluk iddiaları hukuki/kurumsal inceleme gerektirir.

31. Frontend Information Architecture

Ana menü:

Overview
Endpoints
Live Requests
Security
Replay Lab
Mock Lab
Contracts
Agents
Policies
Settings
31.1 Overview

Kartlar:

Requests
Errors
Security Findings
Blocked Requests
Duplicate Events
Average Latency

Grafikler:

requests over time
findings by severity
endpoint error rate
top failing endpoints
31.2 Live Requests

Sol:

request list

Sağ:

inspector

Filtre:

method
endpoint
status
severity
date
source
finding category
31.3 Inspector

Tabs:

Overview
Headers
Query
Payload
Security
Schema
Timeline
Replay
31.4 Security

Finding list:

CRITICAL
HIGH
MEDIUM
LOW
INFO

Her finding:

category
severity
field
masked evidence
explanation
recommendation
action
timestamp
31.5 Replay Lab
Original Request
Target
Headers
Body
Environment
Run

Sonuç:

HTTP status
latency
response
comparison
31.6 Mock Lab

Visual rule builder.

Örnek:

IF
event == payment.failed


THEN
status = 503
delay = 3000ms
body = {...}
32. Realtime

MVP için SSE önerilir.

Akış:

Webhook
   ↓
Backend
   ↓
Redis
   ↓
Event broadcaster
   ↓
SSE
   ↓
Next.js

Frontend:

GET /api/projects/{id}/events/stream

SSE event type:

request.created
request.analyzed
finding.created
replay.completed
agent.status

WebSocket ancak iki yönlü sürekli iletişim gerçekten gerektiğinde kullanılmalıdır.

Agent channel için WebSocket veya secure streaming connection kullanılabilir.

33. REST API Taslağı
Authentication
POST /api/auth/register
POST /api/auth/login
POST /api/auth/refresh
POST /api/auth/logout
Projects
GET    /api/projects
POST   /api/projects
GET    /api/projects/{id}
PATCH  /api/projects/{id}
DELETE /api/projects/{id}
Endpoints
GET    /api/projects/{projectId}/endpoints
POST   /api/projects/{projectId}/endpoints
GET    /api/endpoints/{id}
PATCH  /api/endpoints/{id}
DELETE /api/endpoints/{id}
Captured Requests
GET /api/endpoints/{id}/requests
GET /api/requests/{id}
Security
GET /api/projects/{id}/findings
GET /api/findings/{id}
PATCH /api/findings/{id}
Replay
POST /api/requests/{id}/replay
GET  /api/replay/{id}
Mock
GET    /api/endpoints/{id}/mock-rules
POST   /api/endpoints/{id}/mock-rules
PATCH  /api/mock-rules/{id}
DELETE /api/mock-rules/{id}
Policy
GET   /api/projects/{id}/policies
PUT   /api/projects/{id}/policies
Agent
POST /api/agents
GET  /api/agents
POST /api/agents/{id}/revoke
34. Error Model

API response format:

{
  "error": {
    "code": "ENDPOINT_NOT_FOUND",
    "message": "Endpoint does not exist.",
    "requestId": "req_123"
  }
}

Hata kodları:

AUTH_REQUIRED
AUTH_INVALID
FORBIDDEN
NOT_FOUND
ENDPOINT_DISABLED
RATE_LIMITED
SIGNATURE_INVALID
POLICY_BLOCKED
SCHEMA_INVALID
INTERNAL_ERROR
35. Observability

ApiSentinel kendisi de gözlemlenebilir olmalıdır.

Log:

structured JSON
request ID
trace ID
tenant ID
endpoint ID
latency
status

Metrics:

request count
request latency
analysis latency
queue depth
worker failures
blocked requests
finding count
replay success rate

İleri aşama:

OpenTelemetry
Prometheus
Grafana
36. Audit Logging

Audit log tutulmalıdır.

Örnek:

USER_LOGIN
PROJECT_CREATED
ENDPOINT_CREATED
POLICY_CHANGED
RAW_PAYLOAD_VIEWED
REPLAY_EXECUTED
MOCK_RULE_CHANGED
AGENT_CONNECTED
AGENT_REVOKED
FINDING_RESOLVED

Audit log silinmemeli veya ayrı retention politikasına tabi olmalıdır.

37. Güvenlik Gereksinimleri

Zorunlu:

TLS
password hashing
JWT security
refresh token rotation
RBAC
tenant isolation
input validation
output encoding
SQL parameterization
rate limiting
CORS policy
CSRF protection gerektiği yerlerde
secret encryption
audit logs
sensitive data masking
secure headers
dependency scanning
38. SSRF Koruması

Replay ve upstream URL özellikleri SSRF riski taşır.

Kullanıcı:

http://169.254.169.254

gibi internal metadata endpointlerine istek göndermeye çalışabilir.

Bu nedenle Replay/Proxy Engine:

private IP bloklarını engellemeli
localhost'u cloud tarafından doğrudan erişilebilir hedef olarak kabul etmemeli
DNS rebinding risklerini düşünmeli
redirect policy uygulamalı
allowlist desteklemeli

Local Agent bu nedenle önemlidir.

39. Payload Size Limits

Maksimum body size tanımlanmalıdır.

Örnek MVP:

1 MB

İleri aşamada plan bazlı:

1 MB
5 MB
10 MB

Kullanıcıyı sınırsız payload ile sistemi tüketmekten korur.

40. Retention

Örnek:

Free:
24 hours


Developer:
7 days


Team:
30 days


Enterprise:
custom

İlk MVP'de basit:

7 days

uygulanabilir.

Scheduled cleanup worker gerekir.

41. Proje Klasör Yapısı
Backend
backend/
├── src/
│   ├── main/
│   │   ├── java/com/apisentinel/
│   │   │   ├── auth/
│   │   │   ├── organization/
│   │   │   ├── project/
│   │   │   ├── endpoint/
│   │   │   ├── ingestion/
│   │   │   ├── analysis/
│   │   │   │   ├── pii/
│   │   │   │   ├── secret/
│   │   │   │   ├── injection/
│   │   │   │   ├── schema/
│   │   │   │   └── duplicate/
│   │   │   ├── policy/
│   │   │   ├── replay/
│   │   │   ├── mock/
│   │   │   ├── agent/
│   │   │   ├── realtime/
│   │   │   ├── audit/
│   │   │   └── common/
│   │   └── resources/
│   │       ├── application.yml
│   │       └── db/migration/
│   └── test/
├── pom.xml
└── Dockerfile
Frontend
frontend/
├── app/
│   ├── login/
│   ├── dashboard/
│   ├── projects/
│   ├── endpoints/
│   ├── requests/
│   ├── security/
│   ├── replay/
│   ├── mock/
│   ├── contracts/
│   ├── agents/
│   ├── policies/
│   └── settings/
├── components/
├── lib/
├── hooks/
├── types/
└── public/
Agent
agent/
├── cmd/
├── internal/
│   ├── auth/
│   ├── git/
│   ├── scanner/
│   ├── replay/
│   ├── tunnel/
│   └── config/
├── pkg/
├── go.mod
└── README.md
42. MVP Geliştirme Sırası

AI agent aşağıdaki sırayı izlemelidir.

Phase 0 — Repository Bootstrap
monorepo oluştur
backend
frontend
agent
docker-compose
PostgreSQL
Redis
README
Phase 1 — Authentication & Tenant
users
organizations
memberships
JWT
RBAC
project
Phase 2 — Endpoint Ingestion
endpoint creation
dynamic webhook route
request ID
capture
PostgreSQL persistence
basic response
Phase 3 — Realtime Dashboard
SSE
request stream
inspector
filters
Phase 4 — Security Engine

Sırayla:

email
phone
TCKN
credit card + Luhn
API secret
JWT
SQLi heuristic
XSS heuristic
duplicate
Phase 5 — Policy Engine
ALLOW
WARN
ALERT
BLOCK
MASK
Phase 6 — Mock Engine
status code
delay
body
conditional rules
Phase 7 — Replay Engine
external URL
safe SSRF controls
replay result
comparison
Phase 8 — Local Agent
CLI
login
connect
pre-push hook
secret scan
cloud connection
local replay
Phase 9 — Contract Testing
JSON Schema
validation
findings
UI
Phase 10 — AI
masked findings
explanations
remediation
test generation
Phase 11 — Production Hardening
observability
audit logs
retention
encryption
backups
security testing
load testing
43. MVP Acceptance Criteria

MVP "çalışıyor" kabul edilmesi için:

Authentication
 User register olabilir.
 User login olabilir.
 User project oluşturabilir.
 RBAC çalışır.
Webhook
 User endpoint oluşturabilir.
 /hook/{slug} isteği kabul eder.
 Request PostgreSQL'e kaydedilir.
 Request ID üretilir.
 Dashboard request'i gösterir.
 SSE ile canlı güncellenir.
Security
 Email tespit edilir.
 TCKN tespit edilir.
 Credit card Luhn ile doğrulanır.
 API key pattern'i tespit edilir.
 SQLi heuristic çalışır.
 XSS heuristic çalışır.
 Duplicate detection çalışır.
 Findings oluşturulur.
Policy
 BLOCK çalışır.
 WARN çalışır.
 ALERT çalışır.
 MASK çalışır.
Replay
 Request replay edilebilir.
 Replay sonucu kaydedilir.
 SSRF kontrolleri vardır.
Mock
 Status code değiştirilebilir.
 Delay eklenebilir.
 Response body değiştirilebilir.
Agent
 Agent login olabilir.
 Agent connect olabilir.
 Pre-push scan çalışır.
 Secret bulunduğunda push block edilir.
 Cloud'dan local replay alınabilir.
44. Test Stratejisi
Unit Test

Her scanner bağımsız test edilmelidir.

Örnek:

CreditCardDetectorTest
TcknDetectorTest
EmailDetectorTest
SecretDetectorTest
SqlInjectionDetectorTest
XssDetectorTest
DuplicateDetectorTest
Integration Test
HTTP request
  ↓
Gateway
  ↓
Redis
  ↓
Worker
  ↓
PostgreSQL

uçtan uca test edilmelidir.

Security Test

Özellikle:

authentication bypass
tenant isolation
SSRF
SQL injection
XSS
JWT attacks
replay abuse
payload flooding
path traversal
secret leakage

test edilmelidir.

Load Test

Örneğin:

100 req/s
500 req/s
1000 req/s

MVP'de benchmark yapılmalıdır.

45. False Positive Yönetimi

Security scanner'larda false positive önemli bir problemdir.

Her finding:

rule ID
confidence
evidence
severity

tutabilir.

Örnek:

{
  "type": "SQL_INJECTION",
  "severity": "MEDIUM",
  "confidence": 0.72
}

Confidence güvenlik kararının tek belirleyicisi olmamalıdır.

Kullanıcı:

ignore
resolve
false positive
işaretleyebilir.
46. UX Prensipleri

Dashboard geliştirici odaklı olmalıdır.

Kullanıcıya sadece:

ERROR

gösterilmemeli.

Şu format tercih edilmeli:

🔴 Secret Exposure


Detected in:
payload.apiKey


Why it matters:
This value resembles a live API credential.


Recommended action:
Remove the credential and rotate it.


Policy:
BLOCK

Yani her finding:
What + Why + Action

mantığında gösterilmelidir.

47. Örnek Kullanım Senaryosu — Git Secret

Developer:

application.properties
stripe.secret=sk_live_xxx

komutu:

git push

Akış:

pre-push
 ↓
agent
 ↓
scanner
 ↓
SECRET FOUND
 ↓
policy BLOCK
 ↓
exit 1

GitHub'a push gerçekleşmez.

48. Örnek Kullanım Senaryosu — Webhook PII

Stripe-like provider:

POST /hook/payment

Payload:

{
  "customer": {
    "email": "ahmet@example.com"
  }
}

Policy:

EMAIL -> MASK

Backend'e gönderilecek payload policy'ye göre maskelenebilir veya sadece dashboard görünümü maskelenebilir.

Bu davranış ürün tasarımında açıkça belirtilmelidir.

49. Örnek Kullanım Senaryosu — Critical Block

Payload:

{
  "apiKey": "sk_live_xxx"
}

Policy:

SECRET -> BLOCK

Sonuç:

HTTP/1.1 403 Forbidden

ve dashboard:

CRITICAL
Request Blocked
Secret detected

Ancak provider'a uygun status code seçilebilir.

50. Örnek Kullanım Senaryosu — Duplicate
event_id = evt_123

3 saniye içinde tekrar gelir.

Sonuç:

DUPLICATE_EVENT
severity = MEDIUM
action = WARN

Policy BLOCK ise backend'e iletilmez.

51. Örnek Kullanım Senaryosu — Chaos

Kullanıcı:

Mock Rule:
payment.failed
→ 503
→ delay 3000ms

test eder.

Sistem:

Request
 ↓
ApiSentinel
 ↓
3 sec
 ↓
503

Böylece retry/backoff mekanizması test edilir.

52. Rakip ve Alternatif Konumlandırması

ApiSentinel "dünyada ilk webhook/security ürünü" olarak konumlandırılmamalıdır.

Piyasada farklı problemleri çözen ürünler vardır:

GitHub Secret Scanning / Push Protection
Svix
Postman
Sentry
Datadog
API security ürünleri
secret scanning araçları
webhook testing araçları

ApiSentinel'in hedef farklılaşması:

Git Security
+
API/Webhook Security
+
Integration Testing
+
Replay
+
Mock/Chaos
+
Developer Local Agent
+
Central Policy

tek developer workflow içinde birleştirmektir.

Bu farklılaştırma iddiası ayrıca pazar araştırması ile doğrulanmalıdır.

53. Rakip Analizi İçin Araştırma Görevi

AI agent bu proje için "rakip yok" varsayımı yapmamalıdır.

Araştırılması gereken kategoriler:

Git secret scanners
webhook inspection platforms
API testing platforms
API gateways
API security platforms
observability tools
chaos testing tools
developer security platforms

Her rakip için:

Product
Target user
Main feature
Security features
Replay
Mock
Local agent
Git integration
API runtime scanning
PII detection
Contract testing
Pricing
Weakness
Potential overlap

karşılaştırılmalıdır.

54. MVP'de Yapılmaması Gerekenler

İlk sürümde aşağıdakiler ertelenmelidir:

Kafka
Kubernetes
microservice explosion
enterprise SSO
SAML
complex billing
multi-region
advanced ML
custom LLM training
full SIEM
full WAF
production traffic migration platform

Öncelik:
çalışan ürün + net demo + güvenilir mimari

55. Demo Senaryosu

Sunum için önerilen demo:

Step 1

Developer:

apisentinel connect
Step 2

Project oluşturulur:

Payment Service
Step 3

Endpoint:

/hook/payment
Step 4

Normal webhook gönderilir.

Dashboard:

200 OK
Step 5

Hassas veri gönderilir.

{
  "email": "test@example.com",
  "apiKey": "sk_live_xxx"
}

Dashboard:

🔴 CRITICAL
Secret detected

Policy BLOCK ise:

Request blocked
Step 6

Mock oluştur:

payment.failed
→ 503
→ 3 sec
Step 7

Replay.

Step 8

Git secret senaryosu:

git push

Agent:

❌ PUSH BLOCKED
API SECRET DETECTED

Bu demo, ürünün iki ana dünyasını gösterir:

Developer Security
+
Runtime Integration Security
56. AI Coding Agent İçin Çalışma Kuralları

AI agent bu projeyi geliştirirken aşağıdaki kurallara uymalıdır.

Rule 1

Mevcut çalışan kodu gereksiz yere yeniden yazma.

Rule 2

Önce repository yapısını analiz et.

Rule 3

Bir feature eklemeden önce mevcut entity/API/service yapısını kontrol et.

Rule 4

Security-sensitive kodda test yazmadan implementation tamamlanmış kabul edilmez.

Rule 5

Secret, token, password veya PII hiçbir zaman test loglarına plaintext yazılmamalıdır.

Rule 6

Raw request body loglanmamalıdır.

Rule 7

Tenant isolation bütün backend endpointlerinde uygulanmalıdır.

Rule 8

Frontend authorization'a güvenme.

Rule 9

LLM security decision'ın tek kaynağı olamaz.

Rule 10

Replay URL'leri SSRF kontrolünden geçmelidir.

Rule 11

MVP gereksiz complexity ile şişirilmemelidir.

Rule 12

Her büyük mimari değişiklik README veya architecture decision record'a eklenmelidir.

57. Definition of Done

Bir feature ancak aşağıdakiler sağlanıyorsa tamamlanmış kabul edilir:

[ ] Requirement implemented
[ ] Backend validation implemented
[ ] Authorization implemented
[ ] Error handling implemented
[ ] Unit tests added
[ ] Integration test added where required
[ ] Security implications reviewed
[ ] Logging reviewed
[ ] Sensitive data exposure reviewed
[ ] Documentation updated
[ ] Frontend state handled
[ ] API contract documented
58. Geliştirme Yaklaşımı

AI agent tüm sistemi tek seferde üretmeye çalışmamalıdır.

Doğru sıra:

Understand
   ↓
Design
   ↓
Implement small slice
   ↓
Test
   ↓
Review
   ↓
Integrate
   ↓
Next slice

Her aşamada çalışan sistem korunmalıdır.

59. İlk Sprint

İlk sprint'in hedefi:

"Bir kullanıcının endpoint oluşturup gerçek bir webhook'u yakalaması ve dashboard'da canlı görebilmesi."

İlk sprintte sadece:

Auth
Project
Endpoint
Webhook ingestion
PostgreSQL
Redis
SSE
Dashboard

yeterlidir.

Security engine ilk sprintte zorunlu değildir.

60. İkinci Sprint

Hedef:

Security Engine

Sırayla:

Email
TCKN
Credit Card
API Secret
SQLi
XSS
Duplicate
61. Üçüncü Sprint

Hedef:

Policy + Block

Finding
 ↓
Policy
 ↓
ALLOW / WARN / BLOCK / MASK
62. Dördüncü Sprint

Hedef:

Replay + Mock

63. Beşinci Sprint

Hedef:

Local Agent + Git Hook

64. Altıncı Sprint

Hedef:

Contract + Chaos

65. Yedinci Sprint

Hedef:

AI Explanation + Production Hardening

66. Nihai Ürün Akışı

Son ürün:

                    DEVELOPER
                        |
          +-------------+-------------+
          |                           |
          v                           v
     Local Agent                  Git Provider
          |                           |
          v                           v
    Git Security                Cloud Scan
          |                           |
          +-------------+-------------+
                        |
                        v
                  ApiSentinel
                        |
          +-------------+-------------+
          |             |             |
          v             v             v
       API/Git       Webhooks      Local APIs
          |             |             |
          +-------------+-------------+
                        |
                        v
                  Policy Engine
                        |
          +-------------+-------------+
          |             |             |
          v             v             v
       BLOCK         ALLOW          ALERT
                        |
                        v
                 Analysis Pipeline
                        |
          +-------------+-------------+
          |             |             |
          v             v             v
        PII          Secret         Schema
        SQLi         XSS            Duplicate
                        |
                        v
                    PostgreSQL
                        |
                        v
                   Dashboard
                        |
          +-------------+-------------+
          |             |             |
          v             v             v
       Replay         Mock          AI
        Lab           Lab        Explanation
67. Projenin Nihai Değer Önerisi

ApiSentinel:

Geliştiricilerin kod, API ve webhook seviyesinde hassas veri ve güvenlik risklerini geliştirme aşamasından production'a kadar tespit etmesini; riskleri policy'lere göre engellemesini; API entegrasyonlarını replay, mock ve chaos senaryolarıyla güvenli şekilde test etmesini sağlayan birleşik bir Developer Security & Integration Platform'dur.

Kısa ifade:

Detect. Decide. Prevent. Test.

68. AI Agent'a Son Talimat

Bu doküman projenin source-of-truth dokümanıdır.

Ancak AI agent:

burada yazan her şeyi körü körüne uygulamamalı,
mevcut kodu incelemeli,
güvenlik risklerini ayrıca değerlendirmeli,
çelişki görürse daha güvenli ve daha basit çözümü tercih etmeli,
belirsiz bir gereksinimi varsayarak kritik mimari karar vermemeli,
büyük değişikliklerden önce mevcut mimariyi koruyacak minimum değişikliği değerlendirmelidir.

Özellikle:

Security > Correctness > Simplicity > Performance > Feature Count

önceliği korunmalıdır.

Ama MVP'de gereksiz enterprise complexity eklenmemelidir.

69. Başlangıç Komutu

AI coding agent projeye başladığında ilk olarak:

repository ağacını incele
mevcut dosyaları analiz et
kullanılan teknoloji ve dependency'leri tespit et
bu dokümandaki mimariyle mevcut repository'yi karşılaştır
çelişkileri raporla
uygulanacak ilk küçük milestone'u belirle
doğrudan bütün projeyi üretme
önce Phase 0 + Phase 1'in minimum çalışan yapısını oluştur
testleri yaz
çalıştır
sonucu raporla

İlk hedef:

Authenticated user → Project → Endpoint → Webhook capture → PostgreSQL → SSE → Dashboard

Bu akış çalışmadan Security Engine, Replay, Mock, Agent veya AI katmanlarına geçilmemelidir.
"""
path = Path("/mnt/data/ApiSentinel_PROJECT_SPEC.md")
path.write_text(content, encoding="utf-8")
print(path)