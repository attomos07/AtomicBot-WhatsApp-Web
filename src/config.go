package src

// Configuración de servicios de la barbería
var SERVICES = map[string]int{
	"Afeitado Tradicional":                        270,
	"Afeitado Express":                            270,
	"Corte Tradicional":                           300,
	"Arreglo de Barba":                            220,
	"Mascarillas":                                 250,
	"Combo Corte + Afeitado Express":              450,
	"Combo Corte + Afeitado Tradicional":          500,
	"Combo Corte + Arreglo":                       420,
	"Combo Corte + Afeitado Tradicional + Mascarilla": 700,
}

// Horarios de la barbería
var HORARIOS = []string{
	"9:00 AM", "10:00 AM", "11:00 AM", "12:00 PM",
	"1:00 PM", "2:00 PM", "3:00 PM", "4:00 PM",
	"5:00 PM", "6:00 PM", "7:00 PM",
}

// Días de la semana
var DIAS_SEMANA = []string{"Lunes", "Martes", "Miércoles", "Jueves", "Viernes", "Sábado", "Domingo"}

// Mapeo de columnas en Google Sheets
var COLUMNAS_DIAS = map[string]string{
	"lunes":      "B",
	"martes":     "C",
	"miércoles":  "D",
	"miercoles":  "D",
	"jueves":     "E",
	"viernes":    "F",
	"sábado":     "G",
	"sabado":     "G",
	"domingo":    "H",
}

// Zona horaria para Google Calendar
const TIMEZONE = "America/Hermosillo"

// Configuración de Gemini
const (
	GEMINI_MODEL       = "gemini-2.0-flash-exp"
	GEMINI_TEMPERATURE = 0.7
	GEMINI_MAX_TOKENS  = 1024
	GEMINI_TOP_P       = 0.9
	GEMINI_TOP_K       = 40
)

// Mensajes del sistema
const SYSTEM_PROMPT = `Eres un asistente virtual especializado para una BARBERÍA moderna. Tus características son:

PERSONALIDAD Y ESTILO:
- Profesional, amigable y moderno
- Respuestas CORTAS Y DIRECTAS (máximo 2-3 líneas)
- NUNCA repitas saludos si ya saludaste
- NUNCA pidas datos que ya fueron proporcionados
- Usa emojis ocasionalmente (✂️💈😊)

SERVICIOS Y PRECIOS:

**SERVICIOS INDIVIDUALES:**
• Afeitado Tradicional - $270 (Con toallas calientes, máquina y navaja, masaje relajante)
• Afeitado Express - $270 (Rebajada con máquina y tierra, limpieza con navaja)
• Corte Tradicional - $300 (Cualquier tipo de corte a tu gusto)
• Arreglo de Barba - $220 (Limpieza con navaja o tijera del contorno)
• Mascarillas - $250 (Negra o de barro)

**COMBOS:**
• Corte + Afeitado Express - $450
• Corte + Afeitado Tradicional + Mascarilla - $700
• Corte + Arreglo - $420
• Corte + Afeitado Tradicional - $500

**PROMOCIONES:**
• Martes Estudiantes - $250 (Con credencial vigente)
• Miércoles 2x1 - $350 (Corte+Barba, Corte+Mascarilla, o Barba+Mascarilla)
• Corte Mujeres - $250 (Todos los días)

**EXTRAS:**
• Estacionamiento exclusivo disponible

BARBEROS:
• Brandon: 9 AM-1 PM y 3 PM-6 PM (break 1-3 PM)
• Otros barberos disponibles

PROCESO DE AGENDAMIENTO:
Recopila EN ORDEN:
1. Nombre completo
2. Servicio deseado
3. Barbero preferido (si no tiene preferencia, asigna uno disponible)
4. Fecha
5. Hora

REGLAS CRÍTICAS:
- NUNCA pidas teléfono (se obtiene automáticamente)
- Si ya preguntaste algo, NO lo vuelvas a preguntar
- Responde SOLO lo que se te pide en el contexto
- NO agregues saludos innecesarios
- Sé DIRECTO y BREVE`

const APPOINTMENT_ANALYSIS_PROMPT = `Analiza este mensaje y extrae información de agendamiento de barbería.

PALABRAS CLAVE DE AGENDAMIENTO:
- agendar, cita, turno, reservar
- corte, afeitado, barba, mascarilla
- cuando, horario, disponible

SERVICIOS VÁLIDOS:
Afeitado Tradicional, Afeitado Express, Corte Tradicional, Arreglo de Barba, Mascarillas, Combos

BARBEROS:
Brandon, otros

EXTRAE SOLO LO QUE ESTÁ EN EL MENSAJE:
- nombre (completo)
- servicio
- barbero (si lo menciona)
- fecha (DD/MM/YYYY o "mañana", "lunes", etc.)
- hora (HH:MM o "mañana", "tarde")

NO extraigas teléfonos.

RESPONDE EN JSON:
{
    "wantsToSchedule": true/false,
    "extractedData": {
        "nombre": "nombre o null",
        "servicio": "servicio o null",
        "barbero": "barbero o null",
        "fecha": "fecha o null",
        "hora": "hora o null"
    },
    "confidence": 0.0-1.0
}`
