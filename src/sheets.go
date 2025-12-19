package src

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

var sheetsService *sheets.Service
var spreadsheetID string
var sheetsEnabled bool

// InitSheets inicializa el servicio de Google Sheets
func InitSheets() error {
	spreadsheetID = os.Getenv("SPREADSHEETID")
	if spreadsheetID == "" {
		sheetsEnabled = false
		return fmt.Errorf("SPREADSHEETID no configurado")
	}

	// Verificar credenciales
	if _, err := os.Stat("google.json"); os.IsNotExist(err) {
		sheetsEnabled = false
		return fmt.Errorf("archivo google.json no encontrado")
	}

	ctx := context.Background()
	srv, err := sheets.NewService(ctx, option.WithCredentialsFile("google.json"))
	if err != nil {
		sheetsEnabled = false
		return fmt.Errorf("error creando servicio Sheets: %w", err)
	}

	sheetsService = srv
	sheetsEnabled = true

	log.Println("✅ Google Sheets inicializado correctamente")
	return nil
}

// IsSheetsEnabled verifica si Sheets está habilitado
func IsSheetsEnabled() bool {
	return sheetsEnabled
}

// WriteToSheet escribe datos en una posición específica
func WriteToSheet(values [][]interface{}, rangeStr string) error {
	if !sheetsEnabled {
		return fmt.Errorf("Google Sheets no habilitado")
	}

	valueRange := &sheets.ValueRange{
		Values: values,
	}

	_, err := sheetsService.Spreadsheets.Values.Update(
		spreadsheetID,
		rangeStr,
		valueRange,
	).ValueInputOption("USER_ENTERED").Do()

	if err != nil {
		return fmt.Errorf("error escribiendo en Sheets: %w", err)
	}

	log.Printf("✅ Datos escritos en Sheets: %s\n", rangeStr)
	return nil
}

// ReadSheet lee datos de Google Sheets
func ReadSheet(rangeStr string) ([][]interface{}, error) {
	if !sheetsEnabled {
		return nil, fmt.Errorf("Google Sheets no habilitado")
	}

	resp, err := sheetsService.Spreadsheets.Values.Get(spreadsheetID, rangeStr).Do()
	if err != nil {
		return nil, fmt.Errorf("error leyendo Sheets: %w", err)
	}

	return resp.Values, nil
}

// InitializeWeeklyCalendar inicializa el calendario semanal
func InitializeWeeklyCalendar() error {
	if !sheetsEnabled {
		return fmt.Errorf("Google Sheets no habilitado")
	}

	// Verificar si ya existe
	existingData, err := ReadSheet("Sheet1!A1:H1")
	if err == nil && len(existingData) > 0 && len(existingData[0]) > 1 {
		log.Println("ℹ️  Calendario semanal ya existe")
		return nil
	}

	// Crear encabezados
	headers := []interface{}{"Hora"}
	for _, dia := range DIAS_SEMANA {
		headers = append(headers, dia)
	}

	if err := WriteToSheet([][]interface{}{headers}, "Sheet1!A1:H1"); err != nil {
		return err
	}

	// Crear filas de horarios
	var horariosData [][]interface{}
	for _, hora := range HORARIOS {
		row := []interface{}{hora, "", "", "", "", "", "", ""}
		horariosData = append(horariosData, row)
	}

	rangeStr := fmt.Sprintf("Sheet1!A2:H%d", len(HORARIOS)+1)
	if err := WriteToSheet(horariosData, rangeStr); err != nil {
		return err
	}

	log.Println("✅ Calendario semanal inicializado")
	return nil
}

// SaveAppointmentToCalendar guarda una cita en el calendario
func SaveAppointmentToCalendar(data map[string]string) error {
	if !sheetsEnabled {
		log.Println("⚠️  Google Sheets no habilitado, saltando guardado en Sheets")
		// No retornar error, continuar con Calendar
		return nil
	}

	log.Printf("💾 Guardando cita en Sheets: %v\n", data)

	// Convertir fecha a día de semana
	dia, fechaExacta, err := ConvertirFechaADia(data["fecha"])
	if err != nil {
		return fmt.Errorf("error convirtiendo fecha: %w", err)
	}

	// Normalizar hora
	horaNormalizada, err := NormalizarHora(data["hora"])
	if err != nil {
		return fmt.Errorf("error normalizando hora: %w", err)
	}

	// Obtener columna del día
	columna, exists := COLUMNAS_DIAS[dia]
	if !exists {
		return fmt.Errorf("día no válido: %s", dia)
	}

	// Obtener fila de la hora
	fila := GetFilaHora(horaNormalizada)
	if fila == -1 {
		return fmt.Errorf("hora no válida: %s", horaNormalizada)
	}

	// Leer contenido actual
	celdaRango := fmt.Sprintf("Sheet1!%s%d", columna, fila)
	contenidoActual, _ := ReadSheet(celdaRango)

	// Formatear información de la cita
	infoCita := fmt.Sprintf("👤 %s\n📞 %s\n✂️ %s\n💈 Barbero: %s\n📅 %s",
		data["nombre"],
		data["telefono"],
		data["servicio"],
		data["barbero"],
		fechaExacta,
	)

	var contenidoFinal string
	if len(contenidoActual) > 0 && len(contenidoActual[0]) > 0 {
		// Ya hay contenido, agregar
		existente := contenidoActual[0][0].(string)
		contenidoFinal = fmt.Sprintf("%s\n\n---\n\n%s", existente, infoCita)
	} else {
		contenidoFinal = infoCita
	}

	// Guardar en la celda
	if err := WriteToSheet([][]interface{}{{contenidoFinal}}, celdaRango); err != nil {
		return err
	}

	log.Printf("✅ Cita guardada en Sheets: %s a las %s (celda %s)\n", dia, horaNormalizada, celdaRango)
	log.Printf("📅 Fecha de la cita: %s\n", fechaExacta)

	return nil
}

// GetAppointmentsByDay obtiene las citas de un día específico
func GetAppointmentsByDay(dia string) ([]map[string]interface{}, error) {
	if !sheetsEnabled {
		return nil, fmt.Errorf("Google Sheets no habilitado")
	}

	diaLower := NormalizeText(dia)
	columna, exists := COLUMNAS_DIAS[diaLower]
	if !exists {
		return nil, fmt.Errorf("día no válido: %s", dia)
	}

	// Leer toda la columna del día
	rangeStr := fmt.Sprintf("Sheet1!%s2:%s%d", columna, columna, len(HORARIOS)+1)
	data, err := ReadSheet(rangeStr)
	if err != nil {
		return nil, err
	}

	var citas []map[string]interface{}
	for i, row := range data {
		if len(row) > 0 {
			contenido := row[0].(string)
			if contenido != "" {
				citas = append(citas, map[string]interface{}{
					"hora":      HORARIOS[i],
					"contenido": contenido,
					"dia":       dia,
				})
			}
		}
	}

	return citas, nil
}
