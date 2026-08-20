# ApiSentinel

ApiSentinel, webhook ve API trafiğini yakalamak, incelemek ve güvenlik açısından değerlendirmek için geliştirilen bir TypeScript monorepo uygulamasıdır.

## Gereksinimler

- Node.js 20+
- Docker Desktop

## Yerel geliştirme

1. Ortam değişkenlerini hazırlayın: `Copy-Item .env.example .env`
2. `.env` içindeki `JWT_SECRET` değerini benzersiz ve en az 32 karakterlik güvenli bir değerle değiştirin.
3. Altyapıyı başlatın: `npm run docker:up`
4. İlk veritabanı migration'ını uygulayın: `npm run db:migrate`
5. Backend'i başlatın: `npm run dev:backend`
6. Ayrı bir terminalde frontend'i başlatın: `npm run dev:frontend`

Backend healthcheck: `http://localhost:3001/health`  
Frontend: `http://localhost:3000`

## Doğrulama

```powershell
npm run build
npm test
```

## Mevcut kapsam

Phase 0 ve Phase 1 tamamlanma aşamasındadır: monorepo altyapısı, PostgreSQL/Valkey compose tanımı, JWT tabanlı e-posta/şifre kimlik doğrulama, tenant izolasyonu ve proje CRUD bulunmaktadır. Webhook yakalama Phase 2'de eklenecektir.
