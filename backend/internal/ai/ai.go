package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
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
	Provider         string   `json:"provider,omitempty"`
}

type Explainer struct {
	provider   string // "groq", "openai", "local"
	apiKey     string
	model      string
	apiURL     string
	httpClient *http.Client
}

// NewExplainer initializes the AI Explainer.
// Auto-detects GROQ_API_KEY (default: llama-3.3-70b-versatile) or OPENAI_API_KEY (default: gpt-4o-mini).
// Falls back to high-accuracy local expert rulebook if no keys are provided or network fails.
func NewExplainer(apiKey string) *Explainer {
	e := &Explainer{
		httpClient: &http.Client{Timeout: 12 * time.Second},
	}

	groqKey := os.Getenv("GROQ_API_KEY")
	openaiKey := os.Getenv("OPENAI_API_KEY")
	customModel := os.Getenv("AI_MODEL")

	if apiKey != "" {
		if strings.HasPrefix(apiKey, "gsk_") || strings.EqualFold(os.Getenv("AI_PROVIDER"), "groq") {
			e.provider = "groq"
			e.apiKey = apiKey
			e.apiURL = "https://api.groq.com/openai/v1/chat/completions"
			e.model = "llama-3.3-70b-versatile"
		} else {
			e.provider = "openai"
			e.apiKey = apiKey
			e.apiURL = "https://api.openai.com/v1/chat/completions"
			e.model = "gpt-4o-mini"
		}
	} else if groqKey != "" {
		e.provider = "groq"
		e.apiKey = groqKey
		e.apiURL = "https://api.groq.com/openai/v1/chat/completions"
		e.model = "llama-3.3-70b-versatile"
		log.Info().Str("provider", "Groq").Str("model", e.model).Msg("AI Explainer initialized with Groq Cloud (Llama 3.3 70B)")
	} else if openaiKey != "" {
		e.provider = "openai"
		e.apiKey = openaiKey
		e.apiURL = "https://api.openai.com/v1/chat/completions"
		e.model = "gpt-4o-mini"
		log.Info().Str("provider", "OpenAI").Str("model", e.model).Msg("AI Explainer initialized with OpenAI (GPT-4o Mini)")
	} else {
		e.provider = "local"
		log.Info().Msg("AI Explainer running in Local Knowledgebase mode (Set GROQ_API_KEY or OPENAI_API_KEY for dynamic LLM insights)")
	}

	if customModel != "" {
		e.model = customModel
	}

	return e
}

// ExplainFinding generates a structured remediation report using Groq/OpenAI with fallback to local rulebook.
func (e *Explainer) ExplainFinding(ctx context.Context, category, findingType, severity, maskedEvidence, message string) (*Explanation, error) {
	if e.provider == "groq" || e.provider == "openai" {
		exp, err := e.callLLM(ctx, category, findingType, severity, maskedEvidence, message)
		if err == nil && exp != nil {
			exp.Provider = fmt.Sprintf("%s (%s)", strings.ToUpper(e.provider), e.model)
			return exp, nil
		}
		log.Warn().Err(err).Str("provider", e.provider).Msg("LLM call failed, seamlessly falling back to internal security knowledgebase")
	}

	// Fallback to local expert rulebook
	localExp, err := e.localRulebook(category, findingType, severity, maskedEvidence, message)
	if localExp != nil {
		localExp.Provider = "Dahili Güvenlik Kural Motoru"
	}
	return localExp, err
}

type openAIChatRequest struct {
	Model          string                  `json:"model"`
	Messages       []openAIChatMessage     `json:"messages"`
	Temperature    float64                 `json:"temperature"`
	ResponseFormat *openAIResponseFormat   `json:"response_format,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFormat struct {
	Type string `json:"type"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (e *Explainer) callLLM(ctx context.Context, category, findingType, severity, maskedEvidence, message string) (*Explanation, error) {
	systemPrompt := `Sen kıdemli bir API ve Uygulama Güvenliği (AppSec) uzmanısın.
ApiSentinel güvenlik platformunda yakalanan gerçek güvenlik açığını analiz edip yazılım geliştiriciye Türkçe, net, uygulanabilir ve profesyonel bir düzeltme rehberi (remediation guide) hazırla.

ÖNEMLİ KURALLAR:
1. Geliştiricinin yakalanan gerçek verisini (maskeli kanıt: ` + maskedEvidence + `) ve ilgili bağlamı dikkate al.
2. Asla şablon veya alakasız uydurma veri üretme; verilen bulguya ve maskeli değere özel çözüm üret.
3. KESİNLİKLE sadece aşağıdaki JSON şemasında çıktı ver:
{
  "findingType": string,
  "severity": string,
  "title": string (Örn: "Kredi Kartı Verisi Maruziyeti"),
  "rootCause": string (Kök neden açıklaması, Türkçe),
  "impact": string (Saldırganın bu açıkla ne yapabileceği ve olası zarar),
  "remediationSteps": [string, string, string] (Adım adım 3-4 maddelik çözüm yolu),
  "codeSnippet": string (Gerçek maskeli veriyi ve doğru güvenli pratikleri içeren kod örneği),
  "confidenceScore": number (0.80 - 0.99 arası)
}`

	userPrompt := fmt.Sprintf(`Güvenlik Açığı Detayları:
- Kategori: %s
- Bulgu Türü: %s
- Önem Seviyesi (Severity): %s
- Maskelenmiş Gerçek Kanıt: %s
- Tespit Mesajı: %s

Lütfen geliştiricinin bu spesifik bulguyu (%s) nasıl gidereceğini adım adım ve bu veriye uygun kodla açıkla.`, category, findingType, severity, maskedEvidence, message, maskedEvidence)

	reqBody := openAIChatRequest{
		Model: e.model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.1,
		ResponseFormat: &openAIResponseFormat{Type: "json_object"},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", e.apiURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var chatResp openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, err
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("AI provider error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 || chatResp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("empty response from AI model")
	}

	var explanation Explanation
	if err := json.Unmarshal([]byte(chatResp.Choices[0].Message.Content), &explanation); err != nil {
		return nil, fmt.Errorf("failed to parse AI JSON response: %w", err)
	}

	explanation.FindingType = findingType
	explanation.Severity = severity
	if explanation.ConfidenceScore <= 0 {
		explanation.ConfidenceScore = 0.95
	}

	return &explanation, nil
}

func (e *Explainer) localRulebook(category, findingType, severity, maskedEvidence, message string) (*Explanation, error) {
	// Extract actual last digits if available
	lastDigits := "XXXX"
	cleanEvidence := strings.TrimSpace(maskedEvidence)
	if len(cleanEvidence) >= 4 {
		lastDigits = cleanEvidence[len(cleanEvidence)-4:]
	}

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
			CodeSnippet:     fmt.Sprintf("// Kötü Pratik (Sabit Anahtar):\nconst s3 = new AWS.S3({ accessKeyId: '%s' });\n\n// Güvenli Pratik (IAM Role / STS):\nconst s3 = new AWS.S3(); // AWS SDK ortam/rol değişkenlerinden okur", maskedEvidence),
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
			CodeSnippet:     fmt.Sprintf("// Ham kart verisi yerine tokenize referans kullanın:\n{\n  \"payment_method_id\": \"pm_1N4example...\",\n  \"last4\": \"%s\",\n  \"brand\": \"detected_card\"\n}", lastDigits),
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

	case "SQLI", "SQL_INJECTION":
		return &Explanation{
			FindingType: findingType,
			Severity:    severity,
			Title:       "SQL Injection (SQLi) Saldırı Girişimi",
			RootCause:   fmt.Sprintf("Girdi içerisinde SQL sözdizimini değiştirmeye yönelik zararlı payload (%s) tespit edildi.", maskedEvidence),
			Impact:      "Veritabanının tamamen ele geçirilmesi, yetkisiz veri okuma/silme ve kimlik doğrulama bypass riski.",
			RemediationSteps: []string{
				"1. Asla string birleştirme (concatenation) ile SQL sorgusu oluşturmayın.",
				"2. Parametreli sorgular (Prepared Statements) veya güvenli ORM (sqlc, GORM, Prisma) kullanın.",
				"3. Webhook girişlerinde katı veri tipi ve şema doğrulaması uygulayın.",
			},
			CodeSnippet:     "// Kötü Pratik (Güvensiz):\ndb.Query(\"SELECT * FROM users WHERE email = '\" + email + \"'\")\n\n// Güvenli Pratik (Prepared Statement):\ndb.Query(\"SELECT * FROM users WHERE email = $1\", email)",
			ConfidenceScore: 0.99,
		}, nil

	case "XSS":
		return &Explanation{
			FindingType: findingType,
			Severity:    severity,
			Title:       "Cross-Site Scripting (XSS) Girişimi",
			RootCause:   fmt.Sprintf("Girdi içerisinde çalıştırılabilir HTML/JavaScript etiketi (%s) tespit edildi.", maskedEvidence),
			Impact:      "Oturum çerezlerinin (Session Hijacking) çalınması veya kullanıcı adına yetkisiz işlem yapılması.",
			RemediationSteps: []string{
				"1. Kullanıcıdan gelen tüm girdileri HTML encode işleminden geçirin (Context-aware escaping).",
				"2. DOMPurify veya benzeri kütüphanelerle zengin metin alanlarını sanitize edin.",
				"3. Sıkı bir Content Security Policy (CSP) tanımlayın.",
			},
			CodeSnippet:     "// React / Next.js varsayılan olarak escape eder:\n<div>{userInput}</div>\n\n// Tehlikeli (Kaçının):\n<div dangerouslySetInnerHTML={{ __html: userInput }} />",
			ConfidenceScore: 0.98,
		}, nil

	default:
		return &Explanation{
			FindingType: findingType,
			Severity:    severity,
			Title:       strings.ReplaceAll(findingType, "_", " ") + " Güvenlik Tespiti",
			RootCause:   message,
			Impact:      "Güvenlik veya veri gizliliği standartlarını ihlal edebilecek şüpheli veya hassas girdi.",
			RemediationSteps: []string{
				"1. Girdiyi şema ve tür doğrulamasından geçirin (Strict Input Validation).",
				"2. ApiSentinel Policy Engine üzerinden bu uç nokta için MASK veya BLOCK kuralı tanımlayın.",
				"3. İlgili verinin dış servislerle paylaşılma gerekliliğini gözden geçirin.",
			},
			CodeSnippet:     "// API girdilerini katı şema kontrolünden geçirin\nconst schema = z.object({\n  data: z.string().max(255)\n});",
			ConfidenceScore: 0.90,
		}, nil
	}
}
