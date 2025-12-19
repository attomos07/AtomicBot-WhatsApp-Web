package src

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/types"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"google.golang.org/protobuf/proto"

	waProto "go.mau.fi/whatsmeow/binary/proto"
)

var client *whatsmeow.Client

// SetClient configura el cliente global de WhatsApp
func SetClient(c *whatsmeow.Client) {
	client = c
}

// SendMessage envía un mensaje de texto a un chat
func SendMessage(jid types.JID, text string) error {
	if client == nil {
		return fmt.Errorf("cliente no configurado")
	}

	msg := &waProto.Message{
		Conversation: proto.String(text),
	}

	_, err := client.SendMessage(context.Background(), jid, msg)
	return err
}

// NormalizeText normaliza texto quitando acentos y convirtiendo a minúsculas
func NormalizeText(text string) string {
	text = strings.ToLower(text)
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	result, _, _ := transform.String(t, text)
	return strings.TrimSpace(result)
}

// ContainsKeywords verifica si el texto contiene alguna palabra clave
func ContainsKeywords(text string, keywords []string) bool {
	text = NormalizeText(text)
	for _, keyword := range keywords {
		if strings.Contains(text, NormalizeText(keyword)) {
			return true
		}
	}
	return false
}

// IsGreeting verifica si el mensaje es un saludo
func IsGreeting(text string) bool {
	greetings := []string{
		"hola", "buenos dias", "buenas tardes", "buenas noches",
		"hey", "hi", "hello", "saludos", "que tal",
	}
	return ContainsKeywords(text, greetings)
}

// GetDayOfWeek obtiene el día de la semana en español
func GetDayOfWeek(fecha time.Time) string {
	dias := []string{"domingo", "lunes", "martes", "miércoles", "jueves", "viernes", "sábado"}
	return dias[fecha.Weekday()]
}

// ParseFecha parsea una fecha en formato DD/MM/YYYY
func ParseFecha(fechaStr string) (time.Time, error) {
	partes := strings.Split(fechaStr, "/")
	if len(partes) != 3 {
		return time.Time{}, fmt.Errorf("formato de fecha inválido")
	}

	// Formato: DD/MM/YYYY
	layout := "02/01/2006"
	return time.Parse(layout, fechaStr)
}

// FormatFecha formatea una fecha a DD/MM/YYYY
func FormatFecha(fecha time.Time) string {
	return fecha.Format("02/01/2006")
}

// ConvertirFechaADia convierte una fecha a día de la semana
func ConvertirFechaADia(fecha string) (string, string, error) {
	fechaLower := NormalizeText(fecha)

	// Si ya es un día de la semana
	if _, exists := COLUMNAS_DIAS[fechaLower]; exists {
		fechaExacta := CalcularFechaDelDia(fechaLower)
		return fechaLower, fechaExacta, nil
	}

	// Convertir palabras comunes
	conversiones := map[string]string{
		"hoy":              GetDayOfWeek(time.Now()),
		"mañana":           GetDayOfWeek(time.Now().AddDate(0, 0, 1)),
		"pasado mañana":    GetDayOfWeek(time.Now().AddDate(0, 0, 2)),
		"el lunes":         "lunes",
		"el martes":        "martes",
		"el miercoles":     "miércoles",
		"el miércoles":     "miércoles",
		"el jueves":        "jueves",
		"el viernes":       "viernes",
		"el sabado":        "sábado",
		"el sábado":        "sábado",
		"el domingo":       "domingo",
	}

	if dia, exists := conversiones[fechaLower]; exists {
		fechaExacta := CalcularFechaDelDia(dia)
		return dia, fechaExacta, nil
	}

	// Intentar parsear fecha DD/MM/YYYY
	fechaObj, err := ParseFecha(fecha)
	if err == nil {
		diaSemana := GetDayOfWeek(fechaObj)
		return diaSemana, FormatFecha(fechaObj), nil
	}

	return "", "", fmt.Errorf("formato de fecha no reconocido")
}

// CalcularFechaDelDia calcula la fecha exacta del próximo día especificado
func CalcularFechaDelDia(diaSemana string) string {
	hoy := time.Now()
	diasSemanaNum := map[string]time.Weekday{
		"domingo":   time.Sunday,
		"lunes":     time.Monday,
		"martes":    time.Tuesday,
		"miércoles": time.Wednesday,
		"miercoles": time.Wednesday,
		"jueves":    time.Thursday,
		"viernes":   time.Friday,
		"sábado":    time.Saturday,
		"sabado":    time.Saturday,
	}

	diaObjetivo := diasSemanaNum[NormalizeText(diaSemana)]
	diaActual := hoy.Weekday()

	diasHastaObjetivo := int(diaObjetivo - diaActual)
	if diasHastaObjetivo <= 0 {
		diasHastaObjetivo += 7
	}

	fechaObjetivo := hoy.AddDate(0, 0, diasHastaObjetivo)
	return FormatFecha(fechaObjetivo)
}

// NormalizarHora normaliza una hora al formato del calendario
func NormalizarHora(hora string) (string, error) {
	horaLower := NormalizeText(hora)

	// Conversiones comunes
	conversiones := map[string]string{
		"9":                "9:00 AM",
		"10":               "10:00 AM",
		"11":               "11:00 AM",
		"12":               "12:00 PM",
		"13":               "1:00 PM",
		"14":               "2:00 PM",
		"15":               "3:00 PM",
		"16":               "4:00 PM",
		"17":               "5:00 PM",
		"18":               "6:00 PM",
		"19":               "7:00 PM",
		"9 am":             "9:00 AM",
		"10 am":            "10:00 AM",
		"11 am":            "11:00 AM",
		"12 pm":            "12:00 PM",
		"1 pm":             "1:00 PM",
		"2 pm":             "2:00 PM",
		"3 pm":             "3:00 PM",
		"4 pm":             "4:00 PM",
		"5 pm":             "5:00 PM",
		"6 pm":             "6:00 PM",
		"7 pm":             "7:00 PM",
		"9 de la mañana":   "9:00 AM",
		"10 de la mañana":  "10:00 AM",
		"11 de la mañana":  "11:00 AM",
		"12 del dia":       "12:00 PM",
		"1 de la tarde":    "1:00 PM",
		"2 de la tarde":    "2:00 PM",
		"3 de la tarde":    "3:00 PM",
		"4 de la tarde":    "4:00 PM",
		"5 de la tarde":    "5:00 PM",
		"6 de la tarde":    "6:00 PM",
		"7 de la tarde":    "7:00 PM",
		"mañana":           "10:00 AM",
		"tarde":            "3:00 PM",
		"en la mañana":     "10:00 AM",
		"en la tarde":      "3:00 PM",
	}

	if normalized, exists := conversiones[horaLower]; exists {
		return normalized, nil
	}

	// Buscar en horarios disponibles
	for _, h := range HORARIOS {
		if strings.Contains(horaLower, NormalizeText(h)) {
			return h, nil
		}
	}

	return "", fmt.Errorf("hora no válida")
}

// ConvertirHoraA24h convierte hora de formato 12h a 24h
func ConvertirHoraA24h(hora string) (int, int, error) {
	// Extraer números y determinar AM/PM
	var horas, minutos int
	var periodo string

	// Parsear formatos comunes
	if strings.Contains(hora, "AM") || strings.Contains(hora, "PM") {
		_, err := fmt.Sscanf(hora, "%d:%d %s", &horas, &minutos, &periodo)
		if err != nil {
			return 0, 0, err
		}

		if strings.ToUpper(periodo) == "PM" && horas != 12 {
			horas += 12
		} else if strings.ToUpper(periodo) == "AM" && horas == 12 {
			horas = 0
		}
	} else {
		return 0, 0, fmt.Errorf("formato de hora inválido")
	}

	return horas, minutos, nil
}

// GetFilaHora obtiene el número de fila para una hora específica
func GetFilaHora(hora string) int {
	for i, h := range HORARIOS {
		if h == hora {
			return i + 2 // +2 porque la fila 1 son headers
		}
	}
	return -1
}
