# 🛡️ ApiSentinel

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![Next.js](https://img.shields.io/badge/Next.js-14_App_Router-black?style=for-the-badge&logo=next.js)](https://nextjs.org)
[![gRPC](https://img.shields.io/badge/gRPC-Bi--directional_Streaming-244c5a?style=for-the-badge&logo=grpc)](https://grpc.io)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql)](https://www.postgresql.org)
[![Valkey/Redis](https://img.shields.io/badge/Valkey-7.2-DC382D?style=for-the-badge&logo=redis)](https://valkey.io)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED?style=for-the-badge&logo=docker)](https://www.docker.com)
[![License](https://img.shields.io/badge/License-MIT-green.svg?style=for-the-badge)](LICENSE)

**Geliştirici Odaklı Shift-Left API Güvenliği, Webhook Gözlemlenebilirliği ve Entegrasyon Dayanıklılık Platformu**

[Özellikler](#-temel-özellikler) • [Mimari Şemaları](#-sistem-mimarisi--diyagramlar) • [Görsel Arayüz Turu](#-görsel-arayüz-turu--ekran-görüntüleri) • [Hızlı Başlangıç](#-hızlı-başlangıç) • [CLI Ajanı](#-shift-left-cli-ajanı-apisentinel) • [API Referansı](#-api-rotaları-ve-sözleşmeler)

</div>

---

## 📌 Genel Bakış

**ApiSentinel**, modern mikroservislerin ve bulut mimarilerinin karşılaştığı entegrasyon risklerini hem **kaynakta (Shift-Left Git Hook & CLI)** hem de **çalışma anında (Ingestion Gateway & Reverse Proxy)** tespit eden, önleyen ve gözlemleyen uçtan uca bir güvenlik ve dayanıklılık platformudur.

Harici ödeme sağlayıcıları (Stripe, iyzico), SaaS entegrasyonları (GitHub, Shopify, Slack) ve iç mikroservis trafiği üzerindeki hassas veri sızıntılarını, enjeksiyon saldırılarını, webhook teslimat başarısızlıklarını ve şema sapmalarını tek bir birleşik kontrol düzleminden yönetir.

---

## 🏛 Mimari Şemaları & Diyagramlar

### 1. Uçtan Uca Sistem Mimarisi

Aşağıdaki şema, geliştirici ortamından başlayarak Webhook Gateway, Güvenlik Motoru, gRPC Tüneli ve Dashboard arasındaki veri akışını gösterir:

```mermaid
flowchart TB
    subgraph DevEnv["💻 Geliştirici Ortamı (Shift-Left)"]
        GitRepo["Git Repository<br/>(.git/hooks)"]
        LocalAgent["ApiSentinel CLI Agent<br/>(Go / Shannon Entropy)"]
        LocalService["Local Backend Service<br/>(localhost:8080)"]
        GitRepo -->|pre-commit / pre-push| LocalAgent
    end

    subgraph External["🌐 Dış Servisler & Webhook Kaynakları"]
        Stripe["Stripe / iyzico"]
        GitHub["GitHub / GitLab"]
        ThirdParty["Diğer 3rd Party Webhook'lar"]
    end

    subgraph Gateway["🚪 ApiSentinel Ingestion Gateway"]
        Ingress["/hook/{slug} HTTP Endpoint"]
        SSRF["SSRF & IP Guard"]
        HMAC["HMAC Doğrulama<br/>(Stripe, GitHub, Custom)"]
        RateLimit["Rate Limiter<br/>(Valkey Token Bucket)"]
        
        Ingress --> SSRF --> HMAC --> RateLimit
    end

    subgraph SecurityEngine["⚡ Deterministik Güvenlik Motoru"]
        SecretScan["Secret Scanner<br/>(Regex + Shannon Entropy)"]
        PIIScan["PII Masker<br/>(TCKN, CC / Luhn, Email, IBAN)"]
        InjectionScan["Injection Analyzer<br/>(SQLi, XSS, Command Injection)"]
        EnvelopeDup["Duplicate / Idempotency Checker<br/>(Payload Hash)"]
        ContractCheck["Contract & Schema Drift Validator<br/>(JSON Schema / Baseline)"]

        RateLimit --> SecretScan
        SecretScan --> PIIScan
        PIIScan --> InjectionScan
        InjectionScan --> EnvelopeDup
        EnvelopeDup --> ContractCheck
    end

    subgraph DeliveryPlane["📦 Forwarding & Delivery Plane"]
        Forwarder["HTTP Forwarder"]
        Backoff["Exponential Backoff Retry Engine"]
        DLQ["Dead Letter Queue (DLQ)"]
        ReplayEngine["Instant Replay / Test Runner"]
        
        ContractCheck -->|Geçerli İstek| Forwarder
        Forwarder -->|Başarısız| Backoff
        Backoff -->|Maksimum Deneme Aşıldı| DLQ
        DLQ -.->|Manuel / Otomatik Kurtarma| ReplayEngine
    end

    subgraph Tunnels["🔗 gRPC Bi-Directional Tunnel"]
        gRPCServer["gRPC Agent Service<br/>(:50051)"]
        LocalAgent <==>|Mutual Stream Tunnel| gRPCServer
        Forwarder -.->|Local Route| gRPCServer
        gRPCServer -.->|Forward to Localhost| LocalService
    end

    subgraph Storage["💾 Veri & Durum Katmanı"]
        PG[("PostgreSQL 16<br/>Single Source of Truth")]
        ValkeyDB[("Valkey 7.2 / Redis<br/>Rate Limit & Event Buffer")]
    end

    subgraph ControlPlane["🎛️ Control Plane & UI"]
        Dashboard["Next.js 14 Dashboard<br/>(App Router + TailwindCSS)"]
        SSEStream["Realtime SSE Stream<br/>(/events/stream)"]
        AIEngine["AI Root-Cause & Remediation<br/>(Privacy Redacted)"]
        
        Dashboard <--> SSEStream
        Dashboard <--> AIEngine
    end

    Stripe & GitHub & ThirdParty -->|HTTP POST| Ingress
    SecurityEngine -->|Findings & Telemetry| PG
    DeliveryPlane -->|Audit & Delivery Jobs| PG
    RateLimit <--> ValkeyDB
    PG --> SSEStream
```

---

### 2. Shift-Left Git Hook Güvenlik Döngüsü

Kod commit edilmeden veya uzak repository'ye push edilmeden önce yerel ajanın uyguladığı denetim süreci:

```mermaid
sequenceDiagram
    autonumber
    actor Dev as Geliştirici
    participant Git as Git Client
    participant Hook as .git/hooks/pre-push
    participant CLI as ApiSentinel CLI
    participant Cloud as ApiSentinel Cloud (gRPC)

    Dev->>Git: git push origin main
    Git->>Hook: Hook Tetiklendi
    Hook->>CLI: apisentinel scan --staged
    Note over CLI: Değişen dosyalar analiz ediliyor
    CLI->>CLI: Regex Secret Patterns & Entropy Taraması
    CLI->>CLI: PII & Hassas Veri Kontrolü (TCKN, API Key, Token)

    alt 🚨 Kritik Secret Tespit Edildi
        CLI-->>Dev: ❌ HATA: AWS Secret Key / OpenAI Token bulundu!
        CLI-->>Git: Exit Code 1 (Push Engellendi)
        Git-->>Dev: Push işlemi reddedildi
        opt Cloud Sync Açık
            CLI->>Cloud: SyncScanResults(Repo, Branch, Commit, Findings)
            Cloud-->>CLI: Dashboard Bildirimi Oluşturuldu
        end
    else ✅ Temiz Kod
        CLI-->>Git: Exit Code 0 (Temiz)
        Git->>Dev: Push Başarıyla Tamamlandı
    end
```

---

### 3. Webhook Ingestion & Güvenlik Karar Akışı

Gelen bir webhook çağrısının gateway üzerinde geçirdiği filtreleme ve aksiyon adımları:

```mermaid
sequenceDiagram
    autonumber
    participant Provider as Webhook Sağlayıcı (Stripe/GitHub)
    participant GW as Ingestion Gateway (/hook/{slug})
    participant Rate as Valkey Rate Limiter
    participant Sec as Güvenlik Motoru
    participant Target as Upstream API / Local Agent
    participant DB as PostgreSQL & SSE

    Provider->>GW: POST /hook/stripe-payments (Payload + Signature)
    GW->>GW: Endpoint Doğrulama & SSRF IP Filtresi
    GW->>GW: HMAC İmzası Kontrolü (Header & Secret)
    
    alt Geçersiz İmza
        GW-->>Provider: 401 Unauthorized (Geçersiz Webhook İmzası)
    else İmza Doğrulandı
        GW->>Rate: Rate Limit Kontrolü (IP / Token Bucket)
        alt Hız Sınırı Aşıldı
            GW-->>Provider: 429 Too Many Requests
        else Limit Uygun
            GW->>Sec: Payload Taraması (Secret, PII, SQLi/XSS, Schema)
            
            alt Saldırı / Kural İhlali (Policy: BLOCK)
                Sec-->>GW: Policy Violation (SQL Injection tespit edildi)
                GW->>DB: Olayı Logla & Security Alert Üret
                GW-->>Provider: 403 Forbidden (Blocked by Policy)
            else Güvenli / MASK Edilebilir İstek
                opt PII Tespit Edildi (Policy: MASK)
                    Sec->>Sec: Hassas Alanları Maskele (****-****)
                end
                GW->>Target: İsteği İlet (Forwarding)
                Target-->>GW: 200 OK
                GW->>DB: Teslimat Başarılı (Delivery Timeline)
                GW-->>Provider: 200 OK (İşlendi)
            end
        end
    end
```

---

### 4. Güvenilir Teslimat & DLQ Otomatik Kurtarma Akışı

```mermaid
flowchart LR
    Incoming["Gelen İstek<br/>(Onaylandı)"] --> Dispatch["Hedefe Gönderim<br/>(Forwarding)"]
    Dispatch --> Result{"Hedef Cevabı?"}
    
    Result -->|2xx Başarılı| Success["✅ Başarılı Teslimat<br/>(Timeline Güncellendi)"]
    Result -->|5xx / Timeout| Retry["🔁 Exponential Backoff<br/>(Deneme 1, 2, 3...)"]
    
    Retry --> RetryCheck{"Maksimum Deneme<br/>Aşıldı mı?"}
    RetryCheck -->|Hayır| WaitDelay["⏱️ Bekleme Aralığı<br/>(2s, 4s, 8s, 16s)"] --> Dispatch
    RetryCheck -->|Evet| DLQ["💀 Dead Letter Queue (DLQ)<br/>(Karantinaya Alındı)"]
    
    DLQ --> Inspect["🔍 Dashboard İnceleme &<br/>AI Hata Açıklaması"]
    Inspect --> ManualAction{"Yönetici Aksiyonu"}
    ManualAction -->|Tekrar Dene| Replay["⚡ Instant Replay<br/>(/dlq/{id}/retry)"] --> Dispatch
    ManualAction -->|Temizle| Purge["🗑️ DLQ Kaydı Sil"]
```

---

### 5. Şema Sözleşmesi & Drift (Sapma) Algılama Döngüsü

```mermaid
stateDiagram-v2
    [*] --> BaselineActive: Şema Tanımlandı / Canlı Trafikten Öğrenildi
    
    BaselineActive --> Validating: Yeni Webhook Geldi
    
    Validating --> BaselineActive: Şema Uyumlu (Geçerli)
    
    Validating --> DriftDetected: Eksik Alan / Yeni Alan / Tip Uyuşmazlığı
    
    state DriftDetected {
        [*] --> BreakingChange: Kritik Alan Silinmiş (Alert Tetikle)
        [*] --> NonBreakingChange: İsteğe Bağlı Yeni Alan Eklenmiş
    }
    
    DriftDetected --> DashboardReview: Dashboard Üzerinde İnceleme
    DashboardReview --> BaselineActive: Değişikliği Kabul Et (Sürümü Güncelle)
    DashboardReview --> BaselineActive: Sapmayı Reddet (Geliştiriciyi Uyar)
```

---

## 📸 Görsel Arayüz Turu & Ekran Görüntüleri

ApiSentinel kontrol panelinin ve modüllerinin gerçek ekran görüntüleri aşağıda detaylı açıklamalarıyla sunulmuştur:

### 1. Genel Bakış Dashboard (Overview)
Canlı ağ geçidi durumu, teslimat güvenilirlik başarı oranı, Dead Letter Queue (DLQ) kurtarma kuyruğu ve engellenen güvenlik tehditlerinin genel durum özeti:

![Genel Bakış Dashboard](docs/screenshots/01-dashboard-overview.png)

- **Teslimat Güvenilirlik Oranı:** Başarılı/toplam webhook iletim performans metriği (%73.9).
- **DLQ Backlog:** Upstream hedef çöktüğünde veri kaybını önleyen karantina kuyruğu (12 başarısız webhook).
- **Güvenlik Tehditleri Özeti:** 138 kritik tehdidin (AWS Key & SQLi) çalışma anında engellendiğini gösteren sayaç.

---

### 2. Güvenlik Bulguları & Tehdit Analizi (Security Findings & AI)
Yakalana API/Webhook trafiğinde tespit edilen PII (TCKN, Kredi Kartı), gizli anahtarlar ve enjeksiyon saldırılarının tek merkezden yönetimi:

![Güvenlik Bulguları ve AI Çözüm Rehberi](docs/screenshots/02-security-findings-ai.png)

- **Ciddiyet Dağılımı:** Kritik (138), Yüksek Risk (157), Orta/Bilgi (1071).
- **Maskeli Kanıt:** Hassas verinin loglara ve veri tabanına düz metin düşmesini engelleyen yerel maskeleme (`_des*******************key`).
- **AI Çözüm Rehberi:** Tek tıkla ilgili bulgunun kök nedenini, etkisini ve çözüm önerisini sunan entegre asistan.

---

### 3. Canlı İstek Akışı (Request Inspector & SSE Stream)
Gateway üzerinden geçen tüm HTTP çağrılarının Server-Sent Events (SSE) protokolü ile anlık, gecikmesiz incelenmesi:

![Canlı İstek Akışı](docs/screenshots/03-request-inspector.png)

- **Canlı Durum Rozetleri:** 403 Forbidden (Güvenlik ihlali), 200 OK ve MOCKED etiketli istek filtreleri.
- **Detaylı Payload İnceleyici:** İstek gövdesinde yakalanan `aws_key` ve SQL Injection (`"' OR 1=1 --"`) saldırı denemesinin gerçek zamanlı analizi.
- **cURL Dışa Aktarma:** Tek tıkla isteğin cURL veya JSON kopyasını alma imkanı.

---

### 4. Webhook Teslimat Yönetimi & DLQ Kurtarma (Deliveries)
Doğrulanan webhook'ların upstream sunuculara güvenilir iletimi, deneme zaman çizelgesi ve DLQ kurtarma merkezi:

![Webhook Teslimat Yönetimi ve DLQ](docs/screenshots/04-delivery-dlq.png)

- **Teslimat Durumları:** `DEAD_LETTER (DLQ)` ve `DELIVERED` logları.
- **Akıllı İletim Teşhisi:** Upstream sunucunun verdiği 500 hatasının (`SERVER_INTERNAL_ERROR`) kök neden teşhisi ve otomatik çözüm adımı önerisi.
- **Tek Tıkla Replay:** Hedef servis düzeldiğinde başarısız olmuş webhook'u anında yeniden gönderme.

---

### 5. Şema Sözleşmeleri & Drift Tespiti (Contracts)
Webhook JSON yapılarını versiyonlama, OpenAPI baseline'larını yönetme ve canlı trafik sapmalarını (Drift) yakalama:

![Şema Sözleşmeleri ve Drift Tespiti](docs/screenshots/05-schema-contracts-drift.png)

- **JSON Schema (Draft 2020-12):** Katı sözleşme tanımları ve sözleşme formatlayıcı.
- **Hazır Şablonlar:** Tek tıkla Stripe, iyzico veya GitHub payload formatlarını yükleme.
- **Sözleşme Sürüm Geçmişi:** `Sürüm v1 AKTİF` etiketi ile sürüm takibi ve OpenAPI 3.0 içe aktarım desteği.

---

### 6. Mock Lab & Chaos Test Laboratuvarı (Simulator)
Webhook sağlayıcılarını test etmek için özel HTTP yanıtları, hata durumları ve yapay ağ gecikmeleri simüle etme:

![Mock Lab Simülatörü](docs/screenshots/06-mock-chaos.png)

- **Dinamik Simülasyon Yanıtı:** `HTTP 429 Rate Limit Aşıldı` kuralı ve özel JSON hata gövdesi (`TOO_MANY_REQUESTS`, `retry_after_seconds: 60`).
- **cURL Komut Üretici:** Simülasyonu terminalden anında test etmek için otomatik PowerShell / cURL komutu.

---

### 7. Local Geliştirici Ajanları & gRPC Tüneli (CLI Hub)
Geliştirici bilgisayarlarında çalışan CLI ajanlarını, Pre-Commit/Pre-Push hook'larını ve gRPC tünel bağlantılarını yönetme:

![Local Agent ve gRPC Tüneli](docs/screenshots/07-cli-agent-hub.png)

- **gRPC Standby Tüneli:** `apisentinel connect --server localhost:50051 --token <YOUR_API_KEY>` komutu ile yerel servisi bulut ile köprüleme.
- **Git Hook Kurulum Rehberi:** Tek komutla pre-commit veya pre-push kancalarını yerleştirme.
- **Agent API Key Yönetimi:** Güvenli, projeye izole API anahtarı üretimi ve iptal mekanizması.

---

### 8. İnteraktif Swagger API Dokümantasyonu (Swagger UI)
Backend üzerinde yerleşik çalışan OpenAPI 3.0 dokümantasyonu ve test konsolu:

![Swagger API Dokümantasyonu](docs/screenshots/08-swagger-docs.png)

- **Erişim Adresi:** `http://localhost:3001/docs` veya `http://localhost:3001/swagger`.
- **Kapsam:** Sistem sağlık kontrolleri (`/health`), kimlik doğrulama (`/api/auth/*`), projeler, uç noktalar ve genel `/hook/{slug}` rotaları.

---

## ✨ Temel Özellikler

### 1. Shift-Left Geliştirici Güvenliği (CLI & Git Hooks)
- **Pre-Commit / Pre-Push Entegrasyonu:** Tek bir komutla (`apisentinel install-hook`) Git kancalarına yerleşir.
- **Shannon Entropi Analizi:** Rastgele üretilmiş yüksek entropili token'ları (AWS, OpenAI, Stripe, GitHub PAT, Private Key) matematiksel formülle anında yakalar.
- **Sıfır Bağımlılık & Yüksek Hız:** Go dilinde derlenmiş yerel ikili dosya, git commit işlemlerini geciktirmeden milisaniyeler içinde tamamlar.

### 2. Deterministik Çalışma Zamanı Güvenlik Motoru
- **Secret & API Key Taraması:** 30'dan fazla kurumsal servis anahtar formatı için optimize edilmiş regex ve entropi taraması.
- **PII & KVKK / GDPR Koruması:** TCKN (Mod-10/Mod-11 algoritması), Kredi Kartı (Luhn formülü), E-posta, Telefon ve IBAN doğrulaması ve maskelemesi.
- **Enjeksiyon Saldırı Önleme:** SQLi (`UNION SELECT`, `' OR 1=1`), XSS (`<script>`, `onerror=`) ve İşletim Sistemi Komut Enjeksiyonu tespiti.
- **SSRF & Private IP Filtresi:** Özel IP aralıklarına (RFC 1918), loopback ve AWS/GCP metadata uç noktalarına (169.254.169.254) yapılan istekleri bloklar.
- **Webhook HMAC Doğrulaması:** Stripe, GitHub, Shopify ve özel HMAC imzalarını zamanlama saldırılarına karşı güvenli (`crypto/subtle`) olarak doğrular.

### 3. Dayanıklı Webhook Teslimatı & DLQ Kurtarma
- **At-Least-Once Teslimat:** Upstream servis ayakta değilse istekler kaybolmaz; Valkey tabanlı gecikmeli kuyruğa alınır.
- **Akıllı Exponential Backoff:** Jitter destekli üstel gecikme ile hedef sistemleri aşırı yüklemeden yeniden dener.
- **Dead Letter Queue (DLQ):** Maksimum deneme sayısını aşan istekler karantinaya alınır, inceleme sonrası tek tıkla yeniden çalıştırılabilir.

### 4. Şema Sözleşmeleri & Sözleşme Sapması (Drift) Algılama
- **Otomatik Şema Çıkarımı (Inference):** Gelen örnek webhook payload'larını analiz ederek otomatik JSON Schema taslağı üretir.
- **OpenAPI Desteği:** Mevcut OpenAPI spesifikasyonlarını içeri aktararak katı sözleşme doğrulaması yapar.
- **Breaking Change Uyarıları:** Zorunlu bir alanın kaldırılması veya tip değişmesi durumunda geliştiricileri ve takımı önceden uyarır.

### 5. Mock Sunucusu & Chaos Mühendisliği
- **Hata Simülasyonu:** Geliştiricilerin servislerinin 500, 502, 504 veya 429 HTTP yanıtlarında nasıl davrandığını test etmesini sağlar.
- **Gecikme Enjeksiyonu:** Yavaş ağ koşullarını ve timeout durumlarını simüle etmek için milisaniye cinsinden gecikme ekler.

### 6. Gizlilik Öncelikli AI Açıklama Motoru
- **Yerel Maskeleme:** AI servisine gönderilmeden önce tüm gizli anahtarlar ve PII verileri yerel olarak temizlenir.
- **Kök Neden & Çözüm:** Saldırı veya güvenlik açığının kök nedenini, etkisini ve çözüm kod parçacığını açıklar.

---

## 📊 Karşılaştırma & Güvenlik Matrisi

| Güvenlik Yeteneği | Geleneksel Webhook Araçları | Standart API Gateway | ApiSentinel |
| :--- | :---: | :---: | :---: |
| **Shift-Left Git Hook Denetimi** | ❌ Yok | ❌ Yok | ✅ **Var (Pre-Commit / Pre-Push)** |
| **Shannon Entropi Secret Taraması** | ❌ Yok | ❌ Yok | ✅ **Var (Yerel & Gateway)** |
| **Luhn / Mod-10 PII Doğrulama** | ❌ Yok | ⚠️ Basit Regex | ✅ **Var (Tam Algoritmik)** |
| **SSRF & Metadata IP Koruması** | ❌ Yok | ⚠️ Kısmi | ✅ **Var (Sıkı Private Subnet Filtresi)** |
| **HMAC Zamanlama Güvenli Doğrulama** | ⚠️ Manuel Kodlama | ⚠️ Eklenti Gerektirir | ✅ **Var (Yerleşik Sağlayıcı Şablonları)** |
| **DLQ & Tek Tıkla Replay** | ⚠️ Sınırlı | ❌ Ayrı Altyapı İster | ✅ **Var (Görsel Zaman Çizelgesi ile)** |
| **Şema Sapması (Drift) Tespiti** | ❌ Yok | ⚠️ Harici Servis | ✅ **Var (Canlı Trafikten Öğrenen)** |
| **gRPC Yerel Makine Tüneli** | ⚠️ Üçüncü Parti Araç | ❌ Yok | ✅ **Var (Yerleşik Çift Yönlü gRPC)** |

---

## 🚀 Hızlı Başlangıç

### Önkoşullar
- **Go:** 1.25 veya üzeri
- **Node.js:** v20 veya üzeri
- **Docker & Docker Compose:** PostgreSQL 16 ve Valkey servisleri için

### 1. Depoyu Klonlayın ve Ortam Değişkenlerini Ayarlayın
```powershell
git clone https://github.com/myy16/ApiSentinel.git
cd ApiSentinel

# Ortam değişkenlerini kopyalayın
Copy-Item .env.example .env
```

### 2. Altyapı Servislerini Başlatın (Docker Compose)
```powershell
docker compose up -d
```
> Bu komut arka planda **PostgreSQL 16** (`localhost:5432`) ve **Valkey 7.2** (`localhost:6379`) servislerini ayağa kaldırır.

### 3. Backend Sunucusunu Çalıştırın
```powershell
cd backend
go run ./cmd/server
```
- **Backend Port:** `http://localhost:3001`
- **Swagger UI Dokümantasyonu:** `http://localhost:3001/docs` veya `http://localhost:3001/swagger`
- **gRPC Agent Port:** `localhost:50051`

### 4. Frontend Dashboard'u Başlatın
```powershell
cd ../frontend
npm install
npm run dev
```
- **Dashboard UI:** `http://localhost:3000`

---

## 💻 Shift-Left CLI Ajanı (`apisentinel`)

Geliştiricilerin yerel ortamında çalışan bağımsız Go tabanlı CLI aracı:

```powershell
cd agent
go build -o ../bin/apisentinel.exe ./cmd/apisentinel
```

### Temel Komutlar:

#### 1. Çalışma Dizini veya Dosya Tarama
```powershell
# Tüm dizini tara
apisentinel scan --path .

# Yalnızca Git'e eklenmiş (staged) değişiklikleri tara
apisentinel scan --staged
```

#### 2. Git Kancalarını Kurma (Otomasyon)
```powershell
# Push öncesi taramayı zorunlu kıl (Önerilen)
apisentinel install-hook --type pre-push

# Commit öncesi taramayı etkinleştir
apisentinel install-hook --type pre-commit

# Durumu kontrol et
apisentinel status
```

#### 3. Cloud ile Canlı gRPC Tüneli Kurma
```powershell
apisentinel connect --server localhost:50051 --token <PROJE_API_KEY>
```

---

## 📡 API Rotaları ve Sözleşmeler

Backend, Chi Router üzerinde yapılandırılmış RESTful ve SSE uç noktalarından oluşur:

| Rota | Metot | Rol / Yetki | Açıklama |
| :--- | :---: | :---: | :--- |
| `/health` | `GET` | Public | Servis sağlık durumu |
| `/docs` | `GET` | Public | İnteraktif Swagger UI |
| `/hook/{slug}` | `POST` | Public / HMAC | Genel Webhook Gateway alım noktası |
| `/api/auth/login` | `POST` | Public | Kullanıcı girişi & JWT üretimi |
| `/api/projects` | `GET` / `POST` | Authenticated | Projelerin listelenmesi ve oluşturulması |
| `/api/projects/{id}/endpoints` | `GET` / `POST` | Authenticated | Webhook uç noktası yönetimi |
| `/api/endpoints/{id}/webhook-security` | `GET` / `PUT` | Developer | HMAC sağlayıcı ve secret yönetimi |
| `/api/projects/{id}/requests` | `GET` | Authenticated | Canlı istek günlüğü |
| `/api/projects/{id}/findings` | `GET` | Authenticated | Güvenlik motoru bulguları |
| `/api/projects/{id}/findings/stats` | `GET` | Authenticated | Güvenlik istatistikleri ve dağılımı |
| `/api/ai/explain` | `POST` | Authenticated | AI destekli bulgu analizi ve çözüm önerisi |
| `/api/projects/{id}/events/stream` | `GET` | Authenticated | Gerçek zamanlı Server-Sent Events (SSE) akışı |
| `/api/endpoints/{id}/dlq` | `GET` / `DELETE` | Developer / Owner | DLQ kayıtları listeleme ve temizleme |
| `/api/dlq/{id}/retry` | `POST` | Developer | Başarısız webhook çağrısını yeniden çalıştırma |
| `/api/endpoints/{id}/schemas` | `GET` / `POST` | Developer | JSON Schema sözleşme yönetimi |
| `/api/endpoints/{id}/drifts` | `GET` | Developer | Şema sapma kayıtları ve diff inceleme |
| `/api/agents/sessions` | `GET` | Authenticated | Canlı gRPC ajan oturumları |

---

## ⚡ Performans ve Benchmark Sonuçları

ApiSentinel Güvenlik Motoru, yüksek trafikli webhook akışlarında mikro-saniye seviyesinde karar üretecek şekilde optimize edilmiştir:

| Test Edilen Güvenlik Modülü | İşlem Başına Süre | Bellek Tahsisi | Çıkış Kapasitesi |
| :--- | :---: | :---: | :---: |
| **Shannon Entropi + Secret Taraması** | ~1.42 µs | 320 B/op | > 700.000 req/sec |
| **PII & Luhn Kredi Kartı Taraması** | ~0.85 µs | 128 B/op | > 1.100.000 req/sec |
| **SQLi & XSS Enjeksiyon Filtresi** | ~2.10 µs | 512 B/op | > 470.000 req/sec |
| **HMAC-SHA256 Doğrulama** | ~3.60 µs | 256 B/op | > 270.000 req/sec |
| **Valkey Sliding Window Rate Limiter** | ~0.45 ms (ağ dahil)| Minimal | > 50.000 req/sec |

Benchmark testlerini yerel ortamınızda doğrulamak için:
```powershell
cd backend
go test -bench=. ./internal/security/...
```

---

## 📁 Proje Dizin Yapısı

```text
ApiSentinel/
├── agent/                         # Yerel Go CLI Ajanı (Shift-Left)
│   ├── cmd/apisentinel/           # CLI giriş noktası & Cobra komutları
│   └── internal/
│       ├── client/                # gRPC istemcisi & tünel yöneticisi
│       └── git/                   # Git hook kurucu & diff analizcisi
├── backend/                       # Ana Go Backend Servisi
│   ├── cmd/server/                # HTTP & gRPC sunucu başlangıcı
│   ├── internal/
│   │   ├── agent/                 # gRPC tünel yönlendirici & oturumlar
│   │   ├── ai/                    # Sanitized AI analizi & öneri motoru
│   │   ├── database/              # PostgreSQL SQL migrations & sqlc şemaları
│   │   ├── delivery/              # Exponential backoff & DLQ motoru
│   │   ├── forwarding/            # Upstream HTTP iletim servisi
│   │   ├── security/              # Deterministik güvenlik kuralları
│   │   │   ├── hmac/              # Webhook imza doğrulayıcılar
│   │   │   ├── injection/         # SQLi & XSS tespit motoru
│   │   │   ├── pii/               # TCKN, CC, IBAN maskeleme
│   │   │   ├── ratelimit/         # Valkey tabanlı hız sınırlandırıcı
│   │   │   ├── schema/            # JSON Schema & drift algılayıcı
│   │   │   ├── secret/            # Entropi & Regex secret tarayıcı
│   │   │   └── ssrf/              # Private subnet & metadata koruması
│   │   └── transport/
│   │       ├── grpc/              # gRPC API implementasyonları
│   │       └── http/              # Chi router, REST ve SSE işleyicileri
├── frontend/                      # Next.js 14 Web Kontrol Paneli
│   ├── app/                       # App Router sayfaları
│   │   ├── (dashboard)/           # Overview, Security, Requests, DLQ, Schemas...
│   │   └── (auth)/                # Giriş & Kayıt akışları
│   ├── components/                # Yeniden kullanılabilir UI bileşenleri
│   └── contexts/                  # React Query & SSE context yapıları
├── proto/                         # Protocol Buffers sözleşmeleri (.proto)
├── docs/                          # Dokümantasyon & Ekran Görüntüleri
│   └── screenshots/               # README için görsel ekran görüntüleri
├── docker-compose.yml             # Yerel geliştirme ortamı (DB + Cache)
├── docker-compose.production.yml  # Production hazır ortam konfigürasyonu
└── Makefile                       # Otomasyon ve test betikleri
```

---

## 🔒 Güvenlik & KVKK İlkeleri

- **Privacy by Design:** Hiçbir ham kredi kartı numarası, TCKN veya secret veritabanında düz metin (plaintext) olarak saklanmaz.
- **Tek Doğruluk Kaynağı:** Veritabanı şeması ve migration adımları `backend/internal/database/migrations` dizininde versiyonlanır.
- **Timing Attack Koruması:** Webhook imzaları `crypto/subtle.ConstantTimeCompare` fonksiyonu ile doğrulanır.
- **Air-Gapped AI:** AI modellerine istek gövdesi gönderilmeden önce PII ve anahtarlar tamamen redakte edilir; organizasyon ayarlarından AI motoru tamamen kapatılabilir.

---

## 📜 Lisans

Bu proje **MIT Lisansı** ile lisanslanmıştır. Detaylar için [LICENSE](LICENSE) dosyasına göz atabilirsiniz.
