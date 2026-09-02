# ApiSentinel Ürün Stratejisi ve Evrim Planı

Bu belge, ApiSentinel'in mevcut teknik yapısından hareketle ticari değeri yüksek, anlaşılır ve güvenilir bir ürüne dönüşmesi için ürün yönünü tanımlar. Amaç “her türlü API güvenliği sorununu çözen platform” olmak değil; net bir müşteri problemini rakiplerinden daha iyi çözmektir.

## 1. Karar özeti

ApiSentinel'in ilk ticari konumu şu olmalıdır:

> **Kritik webhook entegrasyonlarını güvenli, izlenebilir ve dayanıklı hâle getiren developer-first delivery security platformu.**

Bu konumda ürünün temel vaadi şudur:

> “ApiSentinel'in kabul ettiği webhook olayları doğrulanır, denetlenir, izlenir; upstream teslimatı başarısızsa kaybolmak yerine kurtarılabilir bir iş akışına alınır.”

Bu vaadin teknik olarak gerçekten desteklenmesi gerekir. Bu nedenle pazarlama, güvenlik ve teslimat dayanıklılığı temelinin tamamlanmasından önce “hiçbir isteği asla kaybetmez” gibi mutlak ifadeler kullanmamalıdır.

## 2. Mevcut ürünün doğru tanımı

ApiSentinel şu anda genel ağ trafiğini izleyen şeffaf bir proxy değildir. Bilgisayardaki her projeyi veya API'yi kendiliğinden bulmaz. Trafik, özellikle oluşturulan ApiSentinel endpoint'ine yönlendirilmelidir.

```text
Webhook sağlayıcısı / API istemcisi
             │
             ▼
   ApiSentinel /hook/{slug}
             │
             ├─ İmza ve rate-limit kontrolü
             ├─ PII / Secret tespiti ve redaction
             ├─ SQLi / XSS analizi
             ├─ JSON Schema doğrulaması
             ├─ Policy kararı: ALLOW / BLOCK / MASK
             ├─ Audit, alert ve dashboard kaydı
             └─ Forwarding / retry / outbox / DLQ
                        │
                        ▼
                 Upstream backend
```

Local Agent ise ayrı bir katmandır: geliştirici bilgisayarındaki belirli Git repository'lerinde staged değişiklikleri tarar. Ağdaki API trafiğini izlemez ve diğer klasörlerdeki repository'leri otomatik keşfetmez.

## 3. Neden webhook odaklı ürün?

Webhook'lar ödeme, sipariş, abonelik, müşteri yönetimi, CI/CD, kaynak kod ve e-ticaret entegrasyonlarında kritik olayları taşır. Sorun sadece kötü niyetli payload değildir:

- Yanlış veya geçersiz imzalı webhook kabul edilebilir.
- Aynı ödeme olayı iki kez işlenebilir.
- Backend kısa süreli kesintide olayı kaçırabilir.
- Bir schema değişikliği entegrasyonu sessizce bozabilir.
- PII veya secret gözlemleme araçlarına sızabilir.
- Operasyon ekibi “hangi olay neden başarısız?” sorusuna cevap veremeyebilir.

Bu, güvenlik ve operasyon problemlerinin aynı olay akışında birleştiği yerdir. ApiSentinel'in farkı, yalnızca bir WAF veya yalnızca bir webhook relay olmamak; **doğrulama + güvenlik + teslimat güvenilirliği + gözlemlenebilirlik** katmanlarını birlikte sunmaktır.

OWASP'ın API güvenlik riskleri arasında yetkilendirme, kimlik doğrulama, kaynak tüketimi, iş akışı suistimali, API envanteri ve güvenli olmayan üçüncü taraf API tüketimi bulunur. ApiSentinel'in başlangıç odağı webhook teslimatı olsa da ürün yol haritası bu risklerle uyumlu ilerlemelidir. Kaynak: [OWASP API Security Top 10](https://owasp.org/www-project-api-security/).

## 4. Hedef müşteri ve problem seçimi

### İlk ideal müşteri profili

- Stripe, iyzico, Shopify, GitHub, Supabase veya benzeri webhook sağlayıcılarını kullanan SaaS ekipleri
- Ödeme, sipariş, abonelik veya üyelik olayları işleyen e-ticaret ekipleri
- Ayrı bir güvenlik operasyon ekibi bulunmayan küçük ve orta ölçekli yazılım şirketleri
- Birden fazla müşteri entegrasyonu yöneten B2B platformlar

### İlk aşamada hedeflenmemesi gereken müşteri

- On-premise, SSO/SAML, SIEM, veri yerleşimi ve uzun satın alma süreci isteyen büyük kurumlar
- Tüm ağ trafiğini şeffaf biçimde denetleyecek ürün arayan şirketler
- Tam kapsamlı WAF, API gateway veya SAST/DAST platformu arayan ekipler

### Müşterinin satın aldığı iş

> “Kritik webhook olaylarım güvenli şekilde alınsın, kaybolmasın, tekrar işlenmesin; bir sorun olduğunda da ekibim birkaç dakika içinde nedenini ve çözümünü görsün.”

## 5. Satılabilir değer önerisi

### Ana değer önerisi

> Ödeme ve entegrasyon webhook'larını uygulama kodunuza ulaşmadan doğrulayın, güvenli biçimde yönetin ve teslimat sorunlarını tekrar oynatılabilir şekilde çözün.

### Destekleyici değerler

- Endpoint bazlı HMAC/signature doğrulama
- Duplicate ve idempotency koruması
- Schema enforcement ve kontrat ihlali görünürlüğü
- PII/secret redaction ile daha güvenli audit kaydı
- Retry, outbox ve DLQ ile teslimat kurtarma
- Replay ve test payload'larıyla daha hızlı sorun çözme
- Takımın anlayabileceği delivery timeline ve alert'ler

### Satışta kullanılmaması gereken iddialar

- “Bilgisayarınızdaki tüm API'leri otomatik korur.”
- “Ağınızdaki tüm webhook trafiğini dinler.”
- “Yapay zekâ her saldırıyı yakalar.”
- “Hiçbir istek asla kaybolmaz.”
- “Tam bir WAF/API gateway alternatifi.”

## 6. Ürün sütunları

| Sütun | Kullanıcı sonucu | Öncelik |
|---|---|---|
| Güvenli kabul | Sahte/geçersiz webhook backend'e ulaşmaz | Çok yüksek |
| Güvenilir teslimat | Kabul edilen olay kaybolmaz; retry/DLQ ile kurtarılır | Çok yüksek |
| Operasyon görünürlüğü | Başarısız olayın nedeni ve durumu hemen anlaşılır | Çok yüksek |
| Kolay entegrasyon | Provider seçilerek dakikalar içinde canlıya çıkılır | Çok yüksek |
| Geliştirici deneyimi | Mock, replay, schema ve Git taraması geliştirme hızını artırır | Orta |
| AI yardımı | Bulgular ve olaylar anlaşılır çözüm önerilerine dönüşür | Orta |

## 7. Öncelikli ürün evrimi

### Faz 0 — Güven temeli

Bu faz tamamlanmadan ürün production kritik iş yükleri için agresif biçimde satılmamalıdır.

- Tüm kaynak bazlı tenant ownership kontrolleri: `projectId`, `endpointId`, `requestId`, `dlqId`, `alertId`
- Backend seviyesinde RBAC: OWNER/MEMBER yetki ayrımı
- Forwarding custom header ve alert webhook URL encryption
- Ham payload, header, query, replay ve stream veri politikasının netleştirilmesi
- SSE token güvenliği ve organization izolasyonu
- HMAC doğrulamasının endpoint bazında açık ve zorunlu yapılandırılması
- Outbox/DLQ için restart, lock, retry ve eşzamanlı worker testleri

**Çıkış kriteri:** Bir tenant başka tenant verisine erişemez; outbox işi uygulama yeniden başlasa da kurtarılabilir; secret değerleri API cevaplarında veya loglarda açık görünmez.

### Faz 1 — Webhook Delivery Control Plane

Kullanıcının her gün göreceği asıl ürün ekranı Security Findings değil, delivery operasyon ekranı olmalıdır.

Eklenmesi gerekenler:

- Delivery listesi: alınan, doğrulanan, iletilen, retry bekleyen, DLQ'da olan olaylar
- Olay timeline'ı: alındı → doğrulandı → policy → forwarding denemeleri → sonuç
- Başarı oranı, hata oranı, DLQ backlog ve en çok hata veren endpoint KPI'ları
- Tek tıkla güvenli replay; idempotency override için açık onay
- Alarm kuralları: “5 dakikada 10 delivery failure”, “DLQ backlog > 20”, “signature failure spike”
- Her olay için correlation/request ID üzerinden iz sürme

**Çıkış kriteri:** Operasyon ekibi başarısız bir ödeme webhook'unu dashboard üzerinden teşhis edip yeniden iletebilir.

### Faz 2 — Provider-first onboarding

Kullanıcı “endpoint oluştur” yerine provider seçmelidir:

```text
[ Stripe ] [ iyzico ] [ Shopify ] [ GitHub ] [ Generic ]
```

Her provider paketi şunları getirmelidir:

- Doğru signature header ve doğrulama algoritması
- Timestamp/replay koruma varsayımları
- Hazır JSON Schema veya event örnekleri
- Önerilen retry davranışı
- Test payload generator
- Provider özel delivery rehberi

**Çıkış kriteri:** Bir geliştirici, seçilen sağlayıcı için 10 dakika içinde güvenli endpoint oluşturup test webhook'unu görebilir.

### Faz 3 — API posture ve sözleşme güvenliği

Webhook odağı korunurken API güvenliğine kontrollü genişleme:

- OpenAPI import
- Endpoint envanteri ve sürüm görünürlüğü
- Schema drift: beklenmeyen alan, değişen tip, yeni event türü uyarısı
- Hassas veri alanı politikaları
- Endpoint bazlı rate-limit ve abuse sinyalleri
- API key yaşam döngüsü, expiry ve last-used görünürlüğü
- API gateway'in önüne yönlendirilmiş API trafiği için aynı policy motoru

NIST, API korumasında hem development öncesi hem runtime kontrollerinin birlikte, risk bazlı ele alınmasını önerir. Kaynak: [NIST SP 800-228](https://www.nist.gov/publications/guidelines-api-protection-cloud-native-systems).

### Faz 4 — Developer workflow entegrasyonu

Local Agent ana ürün değil, webhook ürününü tamamlayan bir developer güvenlik katmanı olmalıdır.

- GitHub/GitLab App veya CI entegrasyonu
- PR/commit üzerinde secret ve hassas veri taraması
- Repository, branch, commit, dosya ve satır bazlı kalıcı bulgular
- Panelden repository bazlı hook/agent durumu
- Pre-commit veya pre-push seçimi
- Kolay kurulum: installer veya local desktop agent

**Çıkış kriteri:** Kullanıcı terminale bağımlı kalmadan repository'yi bağlayabilir ve scan sonuçlarını panelde görebilir.

### Faz 5 — AI operasyon asistanı

AI, ana güvenlik karar vericisi olmamalıdır. Deterministik policy motorunun yanında açıklama ve hızlandırma aracı olmalıdır.

İyi AI işleri:

- “Bu webhook neden engellendi?” açıklaması
- “Bu DLQ kaydı nasıl çözülür?” önerisi
- Olay ve hata kümelerini günlük/haftalık özetleme
- JSON Schema taslağı veya değişiklik önerisi
- Risk önceliklendirmesi
- Provider entegrasyon sorunları için çözüm adımları

Kötü AI işi:

- Her istekte LLM çağırarak ALLOW/BLOCK kararı vermek

Bu yaklaşım gecikme, maliyet, veri hassasiyeti ve yanlış pozitif/negatif riski yaratır.

## 8. Temel kullanıcı akışları

### İlk değer alma akışı

```text
Kayıt ol
  → Provider seç
  → Upstream URL ve signing secret gir
  → ApiSentinel URL'ini provider paneline yapıştır
  → Test payload gönder
  → Dashboard'da başarılı delivery gör
  → Canlı moda geç
```

Bu akışın hedef süresi 10 dakikadan az olmalıdır.

### Olay kurtarma akışı

```text
Alert: ödeme webhook'u iletilemedi
  → Delivery detayını aç
  → Hata ve deneme geçmişini gör
  → Backend sorunu çözüldükten sonra Replay/Retry seç
  → Delivery başarıyla tamamlandı durumunu doğrula
```

### Schema değişikliği akışı

```text
Provider yeni alan gönderdi
  → ApiSentinel schema drift bulgusu üretti
  → Ekip örnek payload ve etkilenen endpoint'i gördü
  → Schema güncellendi veya alan policy ile kabul edildi
```

## 9. Ölçülmesi gereken ürün metrikleri

### Güvenilirlik metrikleri

- Delivery success rate
- DLQ backlog
- Mean time to recovery
- Retry ile kurtarılan delivery oranı
- Duplicate forwarding'in engellenme sayısı
- Outbox işlerinin gecikme süresi

### Güvenlik metrikleri

- Geçersiz signature reddi
- Schema ihlali sayısı
- PII/secret redaction sayısı
- Engellenen injection denemeleri
- Tenant isolation veya authorization reddi

### Ürün metrikleri

- İlk endpoint'i başarıyla kurma süresi
- İlk başarılı delivery'ye kadar geçen süre
- Haftalık aktif endpoint sayısı
- Replay/DLQ çözüm oranı
- Provider template kullanım oranı
- Deneme hesabından ücretli plana dönüşüm

## 10. Paketleme ve fiyatlama hipotezi

Fiyatlama, AI token sayısından çok endpoint, event hacmi, retention ve operasyon özellikleriyle ilişkili olmalıdır.

| Paket | Hedef | Önerilen kapsam |
|---|---|---|
| Free / Developer | Tek geliştirici | Az sayıda endpoint, sınırlı event/retention, temel DLQ |
| Pro | Küçük SaaS/e-ticaret | Daha fazla endpoint, alert, replay, uzun retention, provider paketleri |
| Team | Birden çok ekip | RBAC, ortak dashboard, gelişmiş alert ve audit |
| Enterprise | Büyük kurum | SSO/SAML, SIEM, veri yerleşimi, özel retention, SLA ve destek |

Fiyatlandırma uygulanmadan önce 10–15 hedef kullanıcıyla şu doğrulanmalıdır:

- Hangi failure senaryosu onlar için en pahalı?
- Webhook kaybı veya duplicate işleme maliyeti nedir?
- Mevcut çözümde en çok zaman nerede harcanıyor?
- Hangi özellik için doğrudan ödeme yaparlar?

## 11. Rekabetten ayrışma

ApiSentinel aşağıdaki alanlarda rekabet etmeye çalışmamalıdır:

- Genel CDN/WAF performansı
- Tüm şirket ağı için transparent proxy
- Çok geniş enterprise API management
- Her dil için tam SAST ürünü

ApiSentinel'in ayrışacağı alan:

> **Webhook teslimatının güvenlik, doğruluk ve operasyon görünürlüğünü tek bir geliştirici deneyiminde birleştirmek.**

Bu yaklaşım ayrıca AI ajanları ve API'ler arasındaki artan entegrasyon ihtiyacına da uyumludur. API ekosisteminde güvenlik, discovery, gözlemlenebilirlik ve AI-uyumlu entegrasyonlar giderek daha önemli hale gelmektedir. Örnek pazar sinyali: [Postman State of the API 2025](https://www.postman.com/state-of-api/2025/).

## 12. Ürün metni önerileri

### Ana sayfa başlığı

> Kritik webhook'larınızı güvenle alın, doğrulayın ve kaybetmeden yönetin.

### Açıklama

> ApiSentinel endpoint'lerinize gelen webhook isteklerini imza doğrulama, schema kontrolü, hassas veri maskeleme ve güvenlik politikalarıyla denetler. Başarısız backend teslimatlarını retry, outbox ve DLQ ile görünür ve kurtarılabilir hâle getirir.

### Local Agent açıklaması

> Local Agent, bağlı Git repository'lerinde commit/push öncesi secret ve hassas veri sızıntılarını tarayan isteğe bağlı geliştirici aracıdır. API trafiğini izlemez.

## 13. Yapılmaması gerekenler

- Transparent proxy veya bilgisayardaki tüm trafiği yakalama hedefiyle başlangıç yapmak
- AI tespiti üzerinden mutlak güvenlik iddiası kullanmak
- Güvenilir outbox/DLQ temeli tamamlanmadan yüksek kritik iş yükü vaat etmek
- Local Agent'ı ana satın alma nedeni olarak konumlandırmak
- Büyük enterprise ihtiyaçlarını ilk sürümde çözmeye çalışmak
- Kullanıcının ilk webhook'unu çalıştırmasını zorlaştıran karmaşık onboarding tasarlamak

## 14. İlk 90 gün için önerilen sıra

### İlk 30 gün: Güven ve doğruluk

1. Tenant ownership/RBAC açıklarını kapat.
2. Secret encryption ve redaction politikasını tamamla.
3. Outbox/DLQ failure, restart ve concurrency testlerini ekle.
4. Delivery durum modelini netleştir: `PENDING → PROCESSING → RETRY_WAIT → SENT / DLQ`.
5. Ürün metinlerini endpoint tabanlı gerçek kapsamla uyumlu hâle getir.

### 31–60 gün: Operasyon değeri

1. Delivery timeline ve delivery dashboard oluştur.
2. DLQ/retry/replay akışını kullanıcı dostu hâle getir.
3. Stripe, iyzico ve GitHub için ilk provider template'lerini hazırla.
4. Alert policy'lerini ekle.
5. İlk 5–10 tasarım ortağıyla kullanım testi yap.

### 61–90 gün: Büyüme hazırlığı

1. OpenAPI/schema drift için ilk versiyonu oluştur.
2. GitHub/CI veya Local Agent dashboard entegrasyonunu tamamla.
3. AI incident explainer'ı gerçek delivery olaylarına bağla.
4. Free/Pro paketini ve kullanım limitlerini test et.
5. Landing page'i “Webhook Delivery Security” konumuyla güncelle.

## 15. Karar kriteri

Yeni bir özellik eklenmeden önce şu sorular sorulmalıdır:

1. Kritik webhook teslimatını daha güvenli veya daha güvenilir kılıyor mu?
2. Kullanıcının hata çözme süresini azaltıyor mu?
3. İlk değer alma süresini kısaltıyor mu?
4. Hedef müşterinin doğrudan ödeme yapacağı bir problemi çözüyor mu?
5. Güvenlik iddiasını teknik olarak daha ispatlanabilir hâle getiriyor mu?

Bu sorulara net “evet” denmiyorsa özellik sonraki faza bırakılmalıdır.
