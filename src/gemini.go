package src

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

var geminiClient *genai.Client
var geminiModel *genai.GenerativeModel
var geminiEnabled bool

// AppointmentAnalysis estructura para análisis de agendamiento
type AppointmentAnalysis struct {
	WantsToSchedule bool               `json:"wantsToSchedule"`
	ExtractedData   map[string]string  `json:"extractedData"`
	Confidence      float64            `json:"confidence"`
}

// InitGemini inicializa el cliente de Gemini AI
func InitGemini() error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		geminiEnabled = false
		return fmt.Errorf("GEMINI_API_KEY no configurada")
	}

	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		geminiEnabled = false
		return fmt.Errorf("error creando cliente Gemini: %w", err)
	}

	geminiClient = client
	geminiModel = client.GenerativeModel(GEMINI_MODEL)

	// Configurar parámetros del modelo
	geminiModel.SetTemperature(GEMINI_TEMPERATURE)
	geminiModel.SetMaxOutputTokens(int32(GEMINI_MAX_TOKENS))
	geminiModel.SetTopP(float32(GEMINI_TOP_P))
	geminiModel.SetTopK(int32(GEMINI_TOP_K))

	geminiEnabled = true
	log.Println("✅ Gemini AI inicializado correctamente")
	return nil
}

// IsGeminiEnabled verifica si Gemini está habilitado
func IsGeminiEnabled() bool {
	return geminiEnabled
}

// Chat función principal para chatear con Gemini
func Chat(promptContext, userMessage, conversationHistory string) (string, error) {
	if geminiClient == nil {
		return "", fmt.Errorf("Gemini no inicializado")
	}

	ctx := context.Background()

	// Construir prompt completo
	fullPrompt := fmt.Sprintf("%s\n\nHISTORIAL:\n%s\n\nCONTEXTO: %s\n\nMENSAJE: %s\n\nRESPUESTA (MÁXIMO 2 LÍNEAS):",
		SYSTEM_PROMPT,
		conversationHistory,
		promptContext,
		userMessage,
	)

	// Generar respuesta
	resp, err := geminiModel.GenerateContent(ctx, genai.Text(fullPrompt))
	if err != nil {
		return "", fmt.Errorf("error generando respuesta: %w", err)
	}

	if resp == nil || len(resp.Candidates) == 0 {
		return "¿Podrías repetir eso?", nil
	}

	// Extraer texto de la respuesta
	var answer strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				answer.WriteString(fmt.Sprintf("%v", part))
			}
		}
	}

	result := strings.TrimSpace(answer.String())

	// Limpiar saludos duplicados
	if strings.Contains(strings.ToLower(conversationHistory), "hola") {
		unwantedPhrases := []string{"¡Hola!", "Hola", "¡Bienvenido!", "Bienvenido"}
		for _, phrase := range unwantedPhrases {
			if strings.HasPrefix(result, phrase) {
				result = strings.TrimSpace(result[len(phrase):])
			}
		}
	}

	// Limitar longitud
	if len(result) > 500 {
		result = result[:450] + "..."
	}

	if result == "" {
		return "¿Podrías repetir eso?", nil
	}

	return result, nil
}

// AnalyzeForAppointment analiza si el mensaje indica intención de agendamiento
func AnalyzeForAppointment(message, conversationHistory string, isCurrentlyScheduling bool) (*AppointmentAnalysis, error) {
	if geminiClient == nil {
		// Fallback sin Gemini
		return fallbackAnalysis(message), nil
	}

	ctx := context.Background()

	// Construir prompt de análisis
	analysisPrompt := fmt.Sprintf("%s\n\nHISTORIAL:\n%s\n\nMENSAJE: \"%s\"\n\n¿YA ESTÁ AGENDANDO?: %v",
		APPOINTMENT_ANALYSIS_PROMPT,
		conversationHistory,
		message,
		isCurrentlyScheduling,
	)

	// Generar análisis
	resp, err := geminiModel.GenerateContent(ctx, genai.Text(analysisPrompt))
	if err != nil {
		log.Printf("⚠️  Error en análisis, usando fallback: %v\n", err)
		return fallbackAnalysis(message), nil
	}

	if resp == nil || len(resp.Candidates) == 0 {
		return fallbackAnalysis(message), nil
	}

	// Extraer respuesta
	var responseText string
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				responseText += fmt.Sprintf("%v", part)
			}
		}
	}

	// Extraer JSON de la respuesta
	jsonStart := strings.Index(responseText, "{")
	jsonEnd := strings.LastIndex(responseText, "}")
	
	if jsonStart == -1 || jsonEnd == -1 {
		log.Printf("⚠️  No se pudo extraer JSON, usando fallback")
		return fallbackAnalysis(message), nil
	}

	jsonStr := responseText[jsonStart : jsonEnd+1]

	// Parsear JSON
	var analysis AppointmentAnalysis
	if err := json.Unmarshal([]byte(jsonStr), &analysis); err != nil {
		log.Printf("⚠️  Error parseando JSON: %v, usando fallback\n", err)
		return fallbackAnalysis(message), nil
	}

	// Asegurar que el mapa esté inicializado
	if analysis.ExtractedData == nil {
		analysis.ExtractedData = make(map[string]string)
	}

	log.Printf("📊 Análisis: wantsToSchedule=%v, confidence=%.2f, data=%v",
		analysis.WantsToSchedule,
		analysis.Confidence,
		analysis.ExtractedData,
	)

	return &analysis, nil
}

// fallbackAnalysis análisis simple sin Gemini
func fallbackAnalysis(message string) *AppointmentAnalysis {
	lowerMessage := strings.ToLower(message)
	keywords := []string{"cita", "agendar", "turno", "reservar", "corte", "afeitado", "barba"}

	wantsToSchedule := false
	for _, keyword := range keywords {
		if strings.Contains(lowerMessage, keyword) {
			wantsToSchedule = true
			break
		}
	}

	return &AppointmentAnalysis{
		WantsToSchedule: wantsToSchedule,
		ExtractedData:   make(map[string]string),
		Confidence:      0.6,
	}
}

// CheckGeminiHealth verifica que Gemini esté funcionando
func CheckGeminiHealth() bool {
	if geminiClient == nil {
		return false
	}

	_, err := Chat("", "test", "")
	return err == nil
}
