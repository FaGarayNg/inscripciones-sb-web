package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"servidor-angular/models"
	"servidor-angular/repository"

	"github.com/gorilla/mux"
	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// AbanderadoHandler handles HTTP requests for the abanderados list.
type AbanderadoHandler struct {
	repo *repository.AbanderadoRepository
}

// NewAbanderadoHandler creates a new abanderado handler.
func NewAbanderadoHandler() *AbanderadoHandler {
	return &AbanderadoHandler{
		repo: repository.NewAbanderadoRepository(),
	}
}

// GetAllAbanderados handles GET /api/abanderados.
func (h *AbanderadoHandler) GetAllAbanderados(w http.ResponseWriter, r *http.Request) {
	abanderados, err := h.repo.FindAll()
	if err != nil {
		http.Error(w, "Error al obtener abanderados", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(abanderados)
}

// ImportAbanderados handles POST /api/admin/abanderados/import.
//
// Las columnas se leen por posición, igual que en la importación de alumnos:
// A = Nombre y apellido, B = DNI, C = Curso. Las tres son obligatorias y el
// DNI se usa para no duplicar abanderados ya cargados.
func (h *AbanderadoHandler) ImportAbanderados(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
		http.Error(w, "Error al leer el archivo", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error al obtener el archivo", http.StatusBadRequest)
		return
	}
	defer file.Close()

	if !strings.HasSuffix(strings.ToLower(header.Filename), ".xlsx") {
		http.Error(w, "Solo se permiten archivos .xlsx", http.StatusBadRequest)
		return
	}

	f, err := excelize.OpenReader(file)
	if err != nil {
		http.Error(w, "Error al abrir el archivo Excel", http.StatusBadRequest)
		return
	}
	defer f.Close()

	// Preferimos la hoja "Hoja1" y, si no existe, usamos la primera.
	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		http.Error(w, "El archivo no tiene ninguna hoja", http.StatusBadRequest)
		return
	}

	sheet := "Hoja1"
	if !containsSheet(sheets, sheet) {
		sheet = sheets[0]
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		http.Error(w, "Error al leer las filas del Excel", http.StatusBadRequest)
		return
	}
	if len(rows) < 1 {
		http.Error(w, "El archivo no tiene filas", http.StatusBadRequest)
		return
	}

	existingDNIs, err := h.repo.FindAllDNIs()
	if err != nil {
		http.Error(w, "Error al verificar abanderados existentes", http.StatusInternalServerError)
		return
	}
	seenDNIs := make(map[string]bool, len(existingDNIs))
	for _, dni := range existingDNIs {
		seenDNIs[dni] = true
	}

	var successCount int
	var errorMessages []string

	for i, row := range rows {
		if i == 0 { // La fila 1 es el encabezado.
			continue
		}
		if isEmptyRow(row) { // Filas separadoras: no son un error.
			continue
		}
		if len(row) < 3 {
			errorMessages = append(errorMessages, fmt.Sprintf("Fila %d: Faltan columnas", i+1))
			continue
		}

		name := strings.TrimSpace(row[0])
		dni := strings.TrimSpace(row[1])
		curso := strings.TrimSpace(row[2])

		if name == "" || dni == "" || curso == "" {
			errorMessages = append(errorMessages, fmt.Sprintf("Fila %d: Faltan campos obligatorios", i+1))
			continue
		}

		if seenDNIs[dni] {
			errorMessages = append(errorMessages, fmt.Sprintf("Fila %d: El DNI %s ya está cargado, se omitió", i+1, dni))
			continue
		}
		seenDNIs[dni] = true

		abanderado := models.Abanderado{Name: name, DNI: dni, Curso: curso}
		if err := h.repo.Create(&abanderado); err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("Fila %d: Error al crear abanderado: %v", i+1, err))
			continue
		}

		successCount++
	}

	// Los errores siempre viajan como lista, nunca null.
	if errorMessages == nil {
		errorMessages = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": successCount,
		"errors":  errorMessages,
	})
}

// DeleteAbanderado handles DELETE /api/admin/abanderados/{id}.
func (h *AbanderadoHandler) DeleteAbanderado(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	if err := h.repo.Delete(id); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Abanderado no encontrado", http.StatusNotFound)
			return
		}
		http.Error(w, "Error al eliminar abanderado", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
