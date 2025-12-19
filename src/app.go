package src

import (
	"fmt"
	"log"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types/events"
)

// UserState estado del usuario
type UserState struct {
	IsScheduling        bool
	Step                int
	Data                map[string]string
	ConversationHistory []string
	LastMessageTime     int64
	AppointmentSaved    bool
}

var (
	userStates = make(map[string]*UserState)
	stateMutex sync.RWMutex
)

// GetUserState obtiene o crea el estado de un usuario
func GetUserState(userID string) *UserState {
	stateMutex.Lock()
	defer stateMutex.Unlock()

	if state, exists := userStates[userID]; exists {
		return state
	}

	state := &UserState{
		IsScheduling:        false,
		Step:                0,
		Data:                make(map[string]string),
		ConversationHistory: []string{},
		LastMessageTime:     time.Now().Unix(),
		AppointmentSaved:    false,
	}

	userStates[userID] = state
	return state
}

// ClearUserState limpia el estado de un usuario
func ClearUserState(userID string) {
	stateMutex.Lock()
	defer stateMutex.Unlock()
	delete(userStates, userID)
}

// HandleMessage maneja los mensajes entrantes
func HandleMessage(msg *events.Message, client *whatsmeow.Client) {
	// Ignorar mensajes propios
	if msg.Info.IsFromMe {
		return
	}

	// Ignorar mensajes de grupos
	if msg.Info.IsGroup {
		return
	}

	sender := msg.Info.Sender.User
	senderName := msg.Info.PushName
	if senderName == "" {
		senderName = "Cliente"
	}

	// Obtener texto del mensaje
	var messageText string
	if msg.Message.GetConversation() != "" {
		messageText = msg.Message.GetConversation()
	} else if msg.Message.GetExtendedTextMessage() != nil {
		messageText = msg.Message.GetExtendedTextMessage().GetText()
	}

	if messageText == "" {
		return
	}

	log.Printf("📨 Mensaje de %s (%s): %s\n", senderName, sender, messageText)

	// Procesar mensaje
	response := ProcessMessage(messageText, sender, senderName)

	// Enviar respuesta
	if response != "" {
		if err := SendMessage(msg.Info.Chat, response); err != nil {
			log.Printf("❌ Error enviando mensaje: %v\n", err)
		} else {
			log.Printf("✅ Respuesta enviada a %s\n", senderName)
		}
	}
}

// ProcessMessage procesa un mensaje y genera respuesta
func ProcessMessage(message, userID, userName string) string {
	state := GetUserState(userID)
	state.LastMessageTime = time.Now().Unix()

	log.Printf("📊 Estado actual - isScheduling: %v, appointmentSaved: %v\n",
		state.IsScheduling,
		state.AppointmentSaved,
	)

	// Evitar procesar si ya se guardó recientemente
	if state.AppointmentSaved {
		timeSinceLastMessage := time.Now().Unix() - state.LastMessageTime
		if timeSinceLastMessage < 5 {
			log.Println("⏭️  Mensaje ignorado - cita recién guardada")
			return ""
		}
	}

	// Agregar al historial
	state.ConversationHistory = append(state.ConversationHistory, "Usuario: "+message)

	// Si ya guardó la cita, reiniciar
	if state.AppointmentSaved {
		log.Println("🔄 Reiniciando estado después de cita guardada")
		ClearUserState(userID)
		newState := GetUserState(userID)
		newState.ConversationHistory = append(newState.ConversationHistory, "Usuario: "+message)
		return processNewMessage(message, userID, userName, newState)
	}

	// Analizar intención
	analysis, err := AnalyzeForAppointment(
		message,
		joinHistory(state.ConversationHistory),
		state.IsScheduling,
	)
	if err != nil {
		log.Printf("⚠️  Error en análisis: %v\n", err)
		analysis = &AppointmentAnalysis{
			WantsToSchedule: ContainsKeywords(message, []string{"cita", "agendar"}),
			ExtractedData:   make(map[string]string),
		}
	}

	// Si quiere agendar y no está agendando
	if analysis.WantsToSchedule && !state.IsScheduling {
		return startAppointmentFlow(state, analysis, message, userName)
	}

	// Si está agendando, continuar
	if state.IsScheduling {
		return continueAppointmentFlow(state, analysis, message, userID, userName)
	}

	// Conversación normal
	return handleNormalConversation(message, userName, state)
}

func processNewMessage(message, userID, userName string, state *UserState) string {
	analysis, _ := AnalyzeForAppointment(message, joinHistory(state.ConversationHistory), false)

	if analysis != nil && analysis.WantsToSchedule {
		return startAppointmentFlow(state, analysis, message, userName)
	}

	return handleNormalConversation(message, userName, state)
}

func startAppointmentFlow(state *UserState, analysis *AppointmentAnalysis, message, userName string) string {
	log.Println("🎯 Iniciando proceso de agendamiento")
	state.IsScheduling = true
	state.Step = 1

	// Extraer datos del primer mensaje
	if analysis.ExtractedData != nil {
		for key, value := range analysis.ExtractedData {
			if value != "" && value != "null" {
				state.Data[key] = value
				log.Printf("✅ %s capturado: %s\n", key, value)
			}
		}
	}

	// Determinar qué falta
	missingData := getMissingData(state.Data)
	log.Printf("📊 Datos faltantes: %v\n", missingData)

	var promptContext string
	if len(missingData) > 0 {
		promptContext = fmt.Sprintf("Pide el siguiente dato: %s. Datos ya recopilados: %v. NO pidas teléfono. Sé breve.",
			missingData[0],
			state.Data,
		)
	} else {
		promptContext = "Confirma todos los datos antes de guardar: " + fmt.Sprintf("%v", state.Data)
	}

	response, err := Chat(promptContext, message, joinHistory(state.ConversationHistory))
	if err != nil {
		log.Printf("❌ Error en chat: %v\n", err)
		return "¡Perfecto! Vamos a agendar tu cita. Por favor, dime tu nombre completo:"
	}

	state.ConversationHistory = append(state.ConversationHistory, "Asistente: "+response)
	return response
}

func continueAppointmentFlow(state *UserState, analysis *AppointmentAnalysis, message, userID, userName string) string {
	log.Println("📝 Continuando proceso de agendamiento")

	// Extraer información del mensaje actual
	if analysis.ExtractedData != nil {
		for key, value := range analysis.ExtractedData {
			if value != "" && value != "null" && state.Data[key] == "" {
				state.Data[key] = value
				log.Printf("✅ %s capturado: %s\n", key, value)
			}
		}
	}

	// Verificar datos faltantes
	missingData := getMissingData(state.Data)
	log.Printf("📊 Datos faltantes: %v\n", missingData)
	log.Printf("📋 Datos actuales: %v\n", state.Data)

	if len(missingData) > 0 {
		// Pedir siguiente dato
		promptContext := fmt.Sprintf(
			"Pide ÚNICAMENTE el siguiente dato: %s. Datos ya recopilados: %v. NO repitas preguntas. NO pidas teléfono. Sé breve.",
			missingData[0],
			state.Data,
		)

		response, err := Chat(promptContext, message, joinHistory(state.ConversationHistory))
		if err != nil {
			return fmt.Sprintf("Por favor, dime tu %s:", missingData[0])
		}

		state.ConversationHistory = append(state.ConversationHistory, "Asistente: "+response)
		return response
	}

	// Todos los datos completos - guardar
	return saveAppointment(state, userID, userName)
}

func saveAppointment(state *UserState, userID, userName string) string {
	log.Println("✅ Todos los datos completos - Guardando automáticamente")

	state.AppointmentSaved = true
	telefono := userID

	// Convertir fecha a fecha exacta
	_, fechaExacta, err := ConvertirFechaADia(state.Data["fecha"])
	if err != nil {
		log.Printf("⚠️  Error convirtiendo fecha: %v\n", err)
		fechaExacta = state.Data["fecha"]
	}

	// Normalizar hora
	horaNormalizada, err := NormalizarHora(state.Data["hora"])
	if err != nil {
		log.Printf("⚠️  Error normalizando hora: %v\n", err)
		horaNormalizada = state.Data["hora"]
	}

	appointmentData := map[string]string{
		"nombre":      state.Data["nombre"],
		"telefono":    telefono,
		"servicio":    state.Data["servicio"],
		"barbero":     state.Data["barbero"],
		"fecha":       state.Data["fecha"],
		"fechaExacta": fechaExacta,
		"hora":        horaNormalizada,
	}

	// Guardar en Sheets
	sheetsErr := SaveAppointmentToCalendar(appointmentData)

	// Crear evento en Calendar
	calendarEvent, calendarErr := CreateCalendarEvent(appointmentData)

	// Construir mensaje de confirmación
	confirmation := "¡Perfecto! 🎉 Tu cita ha sido agendada exitosamente.\n\n"
	confirmation += "📋 Resumen de tu cita:\n\n"
	confirmation += fmt.Sprintf("👤 Nombre: %s\n", state.Data["nombre"])
	confirmation += fmt.Sprintf("✂️ Servicio: %s\n", state.Data["servicio"])
	confirmation += fmt.Sprintf("💈 Barbero: %s\n", state.Data["barbero"])
	confirmation += fmt.Sprintf("📅 Fecha: %s\n", fechaExacta)
	confirmation += fmt.Sprintf("⏰ Hora: %s\n\n", horaNormalizada)

	if sheetsErr != nil || calendarErr != nil {
		log.Printf("⚠️  Errores guardando: Sheets=%v, Calendar=%v\n", sheetsErr, calendarErr)
	}

	if calendarEvent != nil {
		confirmation += fmt.Sprintf("🔗 Evento en Calendar: %s\n\n", calendarEvent.HtmlLink)
	}

	confirmation += "Te esperamos en la fecha y hora acordada. ¡Gracias por confiar en nosotros! 😊"

	log.Println("✅ Cita guardada y confirmada")
	return confirmation
}

func handleNormalConversation(message, userName string, state *UserState) string {
	log.Println("💬 Conversación normal")

	promptContext := "Responde de manera amigable. Si el usuario pregunta sobre servicios, precios o promociones, proporciona la información detallada."

	response, err := Chat(promptContext, message, joinHistory(state.ConversationHistory))
	if err != nil {
		// Fallback sin Gemini
		if IsGreeting(message) {
			return fmt.Sprintf("¡Hola %s! ✂️ Soy el asistente virtual de la Barbería.\n\n"+
				"Puedo ayudarte a:\n"+
				"📅 Agendar tu cita\n"+
				"💰 Consultar servicios y precios\n"+
				"🎁 Ver promociones\n\n"+
				"¿En qué puedo asistirte hoy?", userName)
		}
		return "Lo siento, no entendí tu mensaje. ¿Puedes repetirlo?"
	}

	state.ConversationHistory = append(state.ConversationHistory, "Asistente: "+response)
	return response
}

func getMissingData(data map[string]string) []string {
	required := []string{"nombre", "servicio", "barbero", "fecha", "hora"}
	var missing []string

	for _, field := range required {
		if data[field] == "" {
			missing = append(missing, field)
		}
	}

	return missing
}

func joinHistory(history []string) string {
	result := ""
	for _, msg := range history {
		result += msg + "\n"
	}
	return result
}
