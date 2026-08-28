# Ranking Search

Farklı içerik sağlayıcılardan (provider) gelen verileri birleştiren, standart bir puanlama sistemine göre normalize edip sıralayan, arama/filtreleme/sayfalama sunan bir API ve üzerine basit bir dashboard.

## İçindekiler

- [Mimari](#mimari)
- [Teknoloji Tercihleri ve Gerekçeleri](#teknoloji-tercihleri-ve-gerekçeleri)
- [Puanlama Formülü](#puanlama-formülü)
- [API Dokümantasyonu](#api-dokümantasyonu)
- [Kurulum ve Çalıştırma](#kurulum-ve-çalıştırma)
- [Testler](#testler)
- [Bonus / Ekstra Özellikler](#bonus--ekstra-özellikler)
- [Bilinen Sınırlamalar](#bilinen-sınırlamalar)

## Mimari

Katmanlı (hexagonal'a yakın) bir mimari kullanıldı — her katmanın tek bir sorumluluğu var, katmanlar arası bağımlılık interface'ler üzerinden kuruldu (dependency inversion):

```
cmd/api/            → giriş noktası, dependency wiring
internal/
  domain/             → çekirdek veri modelleri (Content, Metrics, ContentType) — davranış içermez
  scoring/             → puanlama formülünün saf implementasyonu
  provider/            → Provider interface + provider factory (config → adapter)
    jsonprovider/       → JSON provider adapter
    xmlprovider/        → XML provider adapter
    httpfetch/           → paylaşılan HTTP fetch mantığı
  repository/          → repository interface'leri (ContentRepository, ProviderRepository)
    postgres/            → Postgres implementasyonu
    cached/              → Redis cache-aside decorator
  cache/               → Cache interface + Redis implementasyonu
  ingest/              → provider'ları çekip normalize edip DB'ye yazan orkestrasyon servisi
  httpapi/             → HTTP handler'lar, router, DTO'lar
web/dashboard/         → go:embed ile binary'ye gömülen statik dashboard (vanilla JS)
migrations/            → SQL şema dosyaları
```

**Veri akışı:** `ingest.Service` periyodik olarak her aktif provider'ı (`providers` tablosundan) çeker → adapter ham veriyi (`JSON`/`XML`) `domain.Content`'e normalize eder → `scoring.Calculate` skoru hesaplar → `ContentRepository.Upsert` ile Postgres'e yazılır. `httpapi`, arama isteklerini `ContentRepository.Search`'e yönlendirir; bu repository önce Redis cache'e bakar (cache-aside), yoksa Postgres'e düşer.

## Teknoloji Tercihleri ve Gerekçeleri

### Backend: Go

Senaryo, iki farklı provider'dan **paralel, rate-limitli** veri çekmeyi gerektiriyor — Go'nun goroutine/channel tabanlı concurrency modeli buna native bir çözüm sunuyor. Framework kullanmayarak (Symfony/Doctrine gibi hazır ORM/DI yerine) mimariyi elle kurdum; bu, her kararı gerekçelendirebilmemi ve kod tabanının "sihirli" davranışlar içermemesini sağladı. Alternatif olarak PHP (Symfony) ya da .NET Core değerlendirildi — ikisi de kurumsal olgunlukları açısından güçlü, ama concurrency modeli Go kadar doğal değil.

### Veritabanı: PostgreSQL

Asıl karmaşıklık veri **dönüşümünde** (provider formatlarını normalize etmek), DB'ye ulaşan veri zaten homojen — bu yüzden MongoDB'nin şema esnekliği avantajı burada gerçek bir problem çözmüyor. Postgres'in native full-text search'ü (`tsvector`/`tsquery`, GIN index) anahtar kelime aramasını ek bir arama motoru (Elasticsearch) olmadan çözüyor; JSONB/array desteği (`tags TEXT[]`) esnekliği koruyor. MySQL de değerlendirildi, ama full-text/array desteği Postgres kadar olgun değil.

### Cache: Redis (cache-aside)

Cache katmanı `cache.Cache` interface'i arkasına alındı — Redis olmadan da (in-memory implementasyonla) sistem çalışabilir, tek satır wiring değişikliğiyle. Redis seçildi çünkü yatay ölçeklenmede (birden fazla API instance'ı) paylaşılan bir cache katmanı gerekiyor; bu karar ucuz bir "sigorta" çünkü interface sayesinde maliyeti düşük. Cache, **fail-open** tasarlandı: Redis erişilemezse istekler loglanıp Postgres'e düşer, sistem hiçbir zaman cache yüzünden kırılmaz — bu, `docker-compose`'da Redis'in `api` için fatal olmayan bir bağımlılık olmasıyla da tutarlı.

### ORM tercih edilmedi

Asıl kritik sorgu (full-text search + dinamik filtre/sıralama/sayfalama + toplam sayı, tek sorguda) bir ORM'in DSL'inin rahat karşılayamadığı bir alan — zaten ham SQL yazmak gerekiyordu. `pgx/v5` (native API, `pgxpool`) doğrudan kullanıldı; `database/sql` uyumlu değil çünkü `context` desteği ve connection pooling'i daha performanslı.

## Puanlama Formülü

```
Final Skor = (Temel Puan * İçerik Türü Katsayısı) + Güncellik Puanı + Etkileşim Puanı

Temel Puan:
  Video → views/1000 + likes/100
  Metin → reading_time + reactions/50

İçerik Türü Katsayısı:  Video: 1.5   |  Metin: 1.0

Güncellik Puanı (yayın tarihine göre):
  1 hafta içinde: +5   |  1 ay içinde: +3   |  3 ay içinde: +1   |  daha eski: +0
  (1 ay = 30 gün, 3 ay = 90 gün olarak sabit gün sayısıyla yaklaşıklandı)

Etkileşim Puanı:
  Video → (likes/views) * 10
  Metin → (reactions/reading_time) * 5
```

İmplementasyon: [`internal/scoring/scorer.go`](internal/scoring/scorer.go), table-driven testlerle her bileşen ayrı doğrulanıyor.

## API Dokümantasyonu

### `GET /api/v1/contents`

Arama, filtreleme, sıralama, sayfalama.

| Parametre | Açıklama | Varsayılan |
|---|---|---|
| `q` | Başlıkta anahtar kelime araması (full-text) | - |
| `type` | `video` veya `text` | - (filtre yok) |
| `sort` | `score` veya `date` | `score` |
| `page` | Sayfa numarası (≥1) | `1` |
| `per_page` | Sayfa başı kayıt (1-100) | `20` |

```json
{
  "data": [
    { "external_id": "a1", "provider": "provider2", "title": "Clean Architecture in Go",
      "type": "text", "score": 298.25, "published_at": "2024-03-14T00:00:00Z",
      "tags": ["programming", "architecture"] }
  ],
  "meta": { "total": 8, "page": 1, "per_page": 20 }
}
```

### `GET /api/v1/contents/{id}`

Tekil içerik detayı. `{id}`, `provider:external_id` formatında (örn. `provider1:v1`). Bulunamazsa `404`.

### `GET /api/v1/health`

Sağlık kontrolü.

## Kurulum ve Çalıştırma

### Docker Compose ile (önerilen)

```bash
cp .env.example .env   # opsiyonel — özelleştirmek istersen; yoksa varsayılan değerlerle çalışır
docker compose up --build
```

Bu tek komut Postgres, Redis ve API'yi ayağa kaldırır, şemayı otomatik kurar (migration dosyaları `docker-entrypoint-initdb.d` ile ilk başlatmada çalışır), ingest servisini başlatır. Dashboard: http://localhost:8080

`.env` dosyası opsiyoneldir — oluşturulmazsa `docker-compose.yml`'deki varsayılan değerler (`.env.example`'daki gibi) kullanılır. Postgres kullanıcı adı/şifresi, DB adı, host port'u ve ingest aralığı `.env` üzerinden özelleştirilebilir.

### Lokal geliştirme

```bash
# Postgres ve Redis'i ayrı çalıştırıp migration'ları elle uygulaman gerekir:
psql $DATABASE_URL -f migrations/001_create_providers.sql
psql $DATABASE_URL -f migrations/002_create_contents.sql

export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/ranking_search"
export REDIS_ADDR="localhost:6379"
export HTTP_ADDR=":8080"
export INGEST_INTERVAL="5m"

go run ./cmd/api
```

### Ortam Değişkenleri

| Değişken | Zorunlu | Açıklama |
|---|---|---|
| `DATABASE_URL` | Evet | Postgres bağlantı string'i, yoksa uygulama başlamaz |
| `REDIS_ADDR` | Hayır | Yoksa/erişilemezse uygulama yine çalışır, cache'siz |
| `HTTP_ADDR` | Hayır | Varsayılan `:8080` |
| `INGEST_INTERVAL` | Hayır | Varsayılan `5m`, geçersizse uyarı loglanıp varsayılana düşülür |

## Testler

```bash
go test ./... -v
```

`scoring` (formülün her bileşeni) ve `provider/jsonprovider`, `provider/xmlprovider` (fixture tabanlı parse testleri, partial-failure senaryoları dahil) için unit testler mevcut.

## Bonus / Ekstra Özellikler

Case study'nin zorunlu kılmadığı, ekstra olarak eklenen özellikler:

- **`GET /api/v1/contents/{id}`** — tekil içerik detay endpoint'i (case study sadece listeleme/arama istiyordu)
- **`providers` tablosu ve operasyonel yönetim** — her provider'ın `enabled`, `rate_limit_rps`, `timeout_ms` değerleri DB'de tutuluyor; deploy yapmadan bir provider'ı açıp kapatmak, rate limit/timeout'unu değiştirmek mümkün
- **Provider sağlık takibi** — `providers.last_status`, `last_error`, `last_fetched_at` ile her provider'ın son fetch durumu izlenebiliyor
- **Redis cache-aside katmanı, fail-open tasarımla** — Redis erişilemese bile sistem doğru çalışmaya devam ediyor, sadece yavaşlıyor; cache soyutlaması sayesinde in-memory implementasyona geçiş tek satırlık bir değişiklik
- **Provider bazlı persistent rate limiting** — `golang.org/x/time/rate` ile token-bucket, her provider için ayrı ve DB'den yapılandırılabilir
- **Partial-failure resilience** — bir provider'ın bir içeriğinde format hatası olsa bile, geri kalan geçerli içerikler kaybolmuyor (`errors.Join` ile hata biriktirme)
- **Full-text search** — `LIKE '%...%'` yerine Postgres'in native `tsvector`/`tsquery` + GIN index'i kullanıldı
- **Yapısal loglama** (`log/slog`) — seviye ve key-value alanlarla, log toplama sistemlerine uygun
- **Graceful shutdown** — `SIGTERM`/`SIGINT` ile hem HTTP sunucusu hem ingest döngüsü temiz şekilde kapanıyor, devam eden istekler için bekleme süresi tanınıyor
- **Dashboard'da debounced arama, XSS-safe render, client-side state yönetimi** — ekstra bir frontend framework/build aracı olmadan
- **`go:embed` ile tek binary dağıtım** — dashboard için ayrı bir statik dosya sunucusu gerekmiyor
- **Docker multi-stage build** — final image sadece derlenmiş binary + CA sertifikaları içeriyor (~15-20MB)

## Bilinen Sınırlamalar

- **Migration'lar** düz SQL dosyaları + `docker-entrypoint-initdb.d` ile çalışıyor, versiyon takibi yapan bir migration aracı (golang-migrate vb.) kullanılmadı — tek seferlik kurulum için yeterli, ama migration geçmişi/rollback desteği yok
- **Arama sonucu cache'i** (`Search`) TTL-only invalidation kullanıyor (60sn), yazma sonrası kesin invalidation yapılmıyor — kombinatoryal arama uzayında hangi cache key'lerin etkilendiğini önceden bilmek pratik değil; bu, en fazla 60 saniyelik bir gecikme (staleness) anlamına geliyor
- **Sayfalama offset-based** (`LIMIT`/`OFFSET`), çok büyük veri setlerinde performans düşebilir — case study'nin ölçeğinde bu bir sorun değil
- **Repository/HTTP/ingest katmanları için unit test yok** — zaman bütçesi önceliği scoring/provider parse mantığına (asıl iş kuralları) verildi; bu katmanlar gerçek Postgres/Redis'e karşı elle uçtan uca test edildi
- **Kimlik doğrulama/yetkilendirme yok** — case study kapsamında istenmedi
