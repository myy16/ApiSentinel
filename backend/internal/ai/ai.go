package ai

import (
	"context"
	"fmt"
	"strings"
)

type Explanation struct {
	FindingType      string   `json:"findingType"`
	Severity         string   `json:"severity"`
	Title            string   `json:"title"`
	RootCause        string   `json:"rootCause"`
	Impact           string   `json:"impact"`
	RemediationSteps []string `json:"remediationSteps"`
	CodeSnippet      string   `json:"codeSnippet"`
	ConfidenceScore  float64  `json:"confidenceScore"`
}

type Explainer struct {
	apiKey string
}

func NewExplainer(apiKey string) *Explainer {
	return &Explainer{apiKey: apiKey}
}

// ExplainFinding generates a structured AI remediation report based STRICTLY on masked evidence
func (e *Explainer) ExplainFinding(ctx context.Context, category, findingType, severity, maskedEvidence, message string) (*Explanation, error) {
	// Built-in intelligent security rulebook & expert knowledgebase
	switch findingType {
	case "AWS_KEY":
		return &Explanation{
			FindingType: findingType,
			Severity:    severity,
			Title:       "AWS Access Key Sızıntısı Tespiti",
			RootCause:   fmt.Sprintf("İstek yükü içerisinde genel kullanıma açık olmaması gereken bir AWS Erişim Anahtarı (%s) tespit edildi.", maskedEvidence),
			Impact:      "Yetkisiz kişilerin bulut altyapınızdaki S3, EC2 veya veritabanı kaynaklarına tam erişim sağlayarak veri sızdırmasına veya maliyet artışına yol açabilir.",
			RemediationSteps: []string{
				"1. AWS IAM Console üzerinden bu erişim anahtarını derhal devre dışı bırakın (Deactivate/Revoke).",
				"2. CloudTrail loglarını inceleyerek anahtarın kullanılıp kullanılmadığını denetleyin.",
				"3. API trafiğinde doğrudan anahtar taşımak yerine AWS STS (AssumeRole) veya IAM Instance Roles kullanın.",
			},
			CodeSnippet:     "// Kötü Pratik (Sabit Anahtar):\nconst s3 = new AWS.S3({ accessKeyId: 'AKIA...' });\n\n// Güvenli Pratik (IAM Role / STS):\nconst s3 = new AWS.S3(); // AWS SDK otomatik ortam/rol değişkenlerinden okur",
			ConfidenceScore: 0.99,
		}, nil

	case "CREDIT_CARD":
		return &Explanation{
			FindingType: findingType,
			Severity:    severity,
			Title:       "PCI-DSS Uyumsuzluğu: Kredi Kartı Numarası Maruziyeti",
			RootCause:   fmt.Sprintf("Webhook payload'u içerisinde Luhn algoritması doğrulanmış geçerli bir kredi kartı numarası (%s) tespit edildi.", maskedEvidence),
			Impact:      "PCI-DSS Regülasyon ihlali ve müşteri finansal verilerinin yetkisiz loglara veya dış servislere sızması riski.",
			RemediationSteps: []string{
				"1. Ham kart numaralarını doğrudan webhook payload'unda göndermeyin.",
				"2. Ödeme sağlayıcısının (Stripe, iyzico) tokenization altyapısını kullanın (Örn: tok_123 veya card_id).",
				"3. Gateway seviyesinde MASK veya BLOCK politikası uygulayın.",
			},
			CodeSnippet:     "// Ham kart verisi yerine tokenize referans kullanın:\n{\n  \"payment_method_id\": \"pm_1N4example...\",\n  \"last4\": \"0366\",\n  \"brand\": \"visa\"\n}",
			ConfidenceScore: 0.98,
		}, nil

	case "TCKN":
		return &Explanation{
			FindingType: findingType,
			Severity:    severity,
			Title:       "KVKK / GDPR İhlali: TC Kimlik Numarası Tespiti",
			RootCause:   fmt.Sprintf("Algoritmik olarak doğrulanmış 11 haneli TCKN verisi (%s) açık metin olarak yakalandı.", maskedEvidence),
			Impact:      "Kişisel verilerin korunması kanunu (KVKK) uyarınca idari para cezaları ve müşteri gizliliği ihlali.",
			RemediationSteps: []string{
				"1. Dış servis entegrasyonlarında TCKN yerine müşteri referans ID'si (UUID) iletin.",
				"2. Saklanması gereken durumlarda veritabanında AES-256 ile şifreleyin ve loglardan maskeleyin.",
			},
			CodeSnippet:     "// Loglama ve iletimde maskeleme fonksiyonu kullanın:\nmaskedTCKN := tckn[:2] + \"*******\" + tckn[9:]",
			ConfidenceScore: 0.99,
		}, nil

	case "GITHUB_TOKEN", "API_KEY":
		return &Explanation{
			FindingType: findingType,
			Severity:    severity,
			Title:       "Hassas API Anahtarı / Token Sızıntısı",
			RootCause:   fmt.Sprintf("Servisler arası iletişimde kullanılmaması gereken gizli bir API belirteci (%s) yakalandı.", maskedEvidence),
			Impact:      "Saldırganların ilgili 3. parti API veya GitHub deposu üzerinde tam yetkili işlem yapmasına olanak tanır.",
			RemediationSteps: []string{
				"1. İlgili sağlayıcı panelinden token'ı geçersiz kılın (Rotate Secret).",
				"2. Token'ları istek gövdesinde değil, sadece güvenli `Authorization: Bearer` başlığında taşıyın.",
				"3. Ortam değişkenlerini HashiCorp Vault veya AWS Secrets Manager'da saklayın.",
			},
			CodeSnippet:     "// .env üzerinden okuyun, koda veya payload'a gömmeyin:\napiKey := os.Getenv(\"STRIPE_SECRET_KEY\")",
			ConfidenceScore: 0.99,
		}, nil

	default:
		return &Explanation{
			FindingType: findingType,
			Severity:    severity,
			Title:       strings.ReplaceAll(findingType, "_", " ") + " Tespiti",
			RootCause:   message,
			Impact:      "Güvenlik veya veri gizliliği politikalarını ihlal edebilecek şüpheli girdi.",
			RemediationSteps: []string{
				"1. Girdiyi şema ve tür doğrulamasından geçirin (Input Validation).",
				"2. Gerekirse ApiSentinel Policy Engine üzerinden MASK veya BLOCK kuralı tanımlayın.",
			},
			CodeSnippet:     "// Girdileri doğrulamak için katı şema kontrolü yapın",
			ConfidenceScore: 0.90,
		}, nil
	}
}
