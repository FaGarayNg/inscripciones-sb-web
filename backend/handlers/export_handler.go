package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"servidor-angular/models"

	"github.com/xuri/excelize/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ExportStudents genera un .xlsx con los alumnos ordenados por la conformación
// de grupos: primero los CUD (resaltados), luego cada grupo con su líder e
// integrantes, y al final los alumnos sin grupo en orden alfabético.
func (h *StudentHandler) ExportStudents(w http.ResponseWriter, r *http.Request) {
	students, err := h.repo.FindAll()
	if err != nil {
		http.Error(w, "Error al obtener alumnos", http.StatusInternalServerError)
		return
	}
	groups, err := h.groupRepo.FindAll()
	if err != nil {
		http.Error(w, "Error al obtener grupos", http.StatusInternalServerError)
		return
	}

	byID := make(map[primitive.ObjectID]models.Student, len(students))
	for _, student := range students {
		byID[student.ID] = student
	}

	// Etiqueta del grupo de cada alumno para la columna "Grupo".
	labels := make(map[primitive.ObjectID]string)

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].CreatedAt.Before(groups[j].CreatedAt)
	})
	for _, group := range groups {
		leader := byID[group.LeaderID]
		if group.Solo {
			labels[group.LeaderID] = "En solitario"
			continue
		}
		labels[group.LeaderID] = "Líder · Grupo de " + leader.Name
		for _, memberID := range group.MemberIDs {
			labels[memberID] = "Integrante · Grupo de " + leader.Name
		}
	}

	var ordered []models.Student
	var important []bool
	seen := make(map[primitive.ObjectID]bool, len(students))

	addStudent := func(student models.Student, highlight bool) {
		if seen[student.ID] {
			return
		}
		seen[student.ID] = true
		ordered = append(ordered, student)
		important = append(important, highlight)
	}

	cud := make([]models.Student, 0)
	rest := make([]models.Student, 0)
	for _, student := range students {
		if isCUD(student.Title) {
			cud = append(cud, student)
		} else {
			rest = append(rest, student)
		}
	}
	sort.Slice(cud, func(i, j int) bool { return cud[i].Name < cud[j].Name })

	for _, student := range cud {
		addStudent(student, true)
	}

	// Los grupos ya vienen ordenados por fecha de creación; dentro de cada uno,
	// líder primero y después sus integrantes en el orden en que fueron aceptados.
	for _, group := range groups {
		if leader, ok := byID[group.LeaderID]; ok {
			addStudent(leader, false)
		}
		for _, memberID := range group.MemberIDs {
			if member, ok := byID[memberID]; ok {
				addStudent(member, false)
			}
		}
	}

	// Los alumnos sin grupo y no CUD cierran el listado en orden alfabético.
	ungrouped := make([]models.Student, 0)
	for _, student := range rest {
		if _, inGroup := labels[student.ID]; !inGroup {
			ungrouped = append(ungrouped, student)
		}
	}
	sort.Slice(ungrouped, func(i, j int) bool { return ungrouped[i].Name < ungrouped[j].Name })
	for _, student := range ungrouped {
		addStudent(student, false)
	}

	f := excelize.NewFile()
	defer f.Close()
	f.SetSheetName("Sheet1", "Hoja1")

	headers := []string{"Nombre", "Email", "Instancia", "Titulo", "Grupo"}
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		http.Error(w, "Error al estilizar el archivo", http.StatusInternalServerError)
		return
	}
	importantStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "#7A5B00"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#FFF3CD"}, Pattern: 1},
	})
	if err != nil {
		http.Error(w, "Error al estilizar el archivo", http.StatusInternalServerError)
		return
	}

	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue("Hoja1", cell, header)
		f.SetCellStyle("Hoja1", cell, cell, headerStyle)
	}
	f.SetRowHeight("Hoja1", 1, 24)

	for index, student := range ordered {
		row := index + 2
		values := []string{
			student.Name,
			student.Email,
			student.Instance,
			student.Title,
			labels[student.ID],
		}
		for i, value := range values {
			cell, _ := excelize.CoordinatesToCellName(i+1, row)
			f.SetCellValue("Hoja1", cell, value)
			if important[index] {
				f.SetCellStyle("Hoja1", cell, cell, importantStyle)
			}
		}
	}

	f.SetColWidth("Hoja1", "A", "A", 38)
	f.SetColWidth("Hoja1", "B", "B", 34)
	f.SetColWidth("Hoja1", "C", "C", 18)
	f.SetColWidth("Hoja1", "D", "D", 30)
	f.SetColWidth("Hoja1", "E", "E", 34)
	f.SetPanes("Hoja1", &excelize.Panes{
		Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft",
	})
	_ = f.AutoFilter("Hoja1", "A1:E"+fmt.Sprintf("%d", len(ordered)+1), nil)

	legend := "Las filas resaltadas en amarillo son ingresantes con título CUD y se listan primero."
	f.NewSheet("Leyenda")
	f.SetCellValue("Leyenda", "A1", legend)
	f.SetColWidth("Leyenda", "A", "A", 105)

	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", `attachment; filename="listado-alumnos.xlsx"`)
	if err := f.Write(w); err != nil {
		http.Error(w, "Error al generar el archivo", http.StatusInternalServerError)
	}
}

// isCUD detecta tanto la sigla como el texto completo del beneficio.
func isCUD(title string) bool {
	value := strings.ToLower(strings.TrimSpace(title))
	return strings.Contains(value, "cud") || strings.Contains(value, "discapacidad")
}
