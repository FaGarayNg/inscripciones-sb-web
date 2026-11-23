// Command gen-abanderados-template genera la plantilla para cargar la lista de
// abanderados y escoltas titulares desde el panel administrativo.
//
// Uso:
//
//	go run ./cmd/gen-abanderados-template
//
// A diferencia de gen-template (que reinyecta validaciones en un archivo ya
// existente), este comando construye el .xlsx desde cero con excelize, así que
// la plantilla es reproducible y no depende de un binario armado a mano.
// Es idempotente: sobrescribe el archivo en cada corrida.
//
// Columnas de Hoja1 (por posición, como en la importación de alumnos):
//
//	A = Nombre y apellido, B = DNI, C = Curso.
//
// Curso es texto libre (por ejemplo "1° A"): no lleva lista desplegable porque
// los cursos pueden cambiar entre ciclos lectivos.
package main

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"

	"github.com/xuri/excelize/v2"
)

const (
	dataSheet = "Hoja1"
	helpSheet = "Instrucciones"
)

var headers = []string{"Nombre y apellido", "DNI", "Curso"}

var instructions = []string{
	"Cómo cargar la lista de abanderados",
	"1. Extensión .xlsx, máximo 10 MB.",
	"2. La hoja debe llamarse Hoja1 (si no existe, se usa la primera).",
	"3. Los datos van en Hoja1, a partir de la fila 2.",
	"4. La fila 1 es el encabezado y se ignora. Las columnas se leen por posición,",
	"   así que el orden importa aunque cambies los textos.",
	"5. Columnas: A = Nombre y apellido, B = DNI, C = Curso. Las tres son obligatorias.",
	"6. El curso es texto libre, por ejemplo: 1° A, 2° B, 3° C.",
	"7. Los DNI repetidos (en la base o entre filas) se omiten y se reportan.",
}

func main() {
	tplPath := templatePath()

	f := excelize.NewFile()
	defer f.Close()

	if err := f.SetSheetName("Sheet1", dataSheet); err != nil {
		log.Fatalf("no se pudo renombrar la hoja de datos: %v", err)
	}

	headerStyle, err := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#1F4E79"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		log.Fatalf("no se pudo crear el estilo de encabezados: %v", err)
	}

	for i, header := range headers {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			log.Fatalf("no se pudo resolver la celda del encabezado: %v", err)
		}
		if err := f.SetCellValue(dataSheet, cell, header); err != nil {
			log.Fatalf("no se pudo escribir el encabezado %s: %v", header, err)
		}
		if err := f.SetCellStyle(dataSheet, cell, cell, headerStyle); err != nil {
			log.Fatalf("no se pudo estilizar el encabezado %s: %v", header, err)
		}
	}

	f.SetColWidth(dataSheet, "A", "A", 30)
	f.SetColWidth(dataSheet, "B", "B", 16)
	f.SetColWidth(dataSheet, "C", "C", 16)

	if _, err := f.NewSheet(helpSheet); err != nil {
		log.Fatalf("no se pudo crear la hoja %s: %v", helpSheet, err)
	}
	for i, line := range instructions {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			log.Fatalf("no se pudo resolver la celda de instrucciones: %v", err)
		}
		if err := f.SetCellValue(helpSheet, cell, line); err != nil {
			log.Fatalf("no se pudo escribir la instrucción %d: %v", i+1, err)
		}
	}
	f.SetColWidth(helpSheet, "A", "A", 100)

	if err := f.SaveAs(tplPath); err != nil {
		log.Fatalf("no se pudo guardar la plantilla: %v", err)
	}

	fmt.Printf("Plantilla de abanderados generada: %s\n", tplPath)
}

// templatePath resuelve la ruta de la plantilla a partir del código fuente, de
// modo que el comando funcione desde cualquier directorio de trabajo.
func templatePath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("no se pudo resolver la ruta de la plantilla")
	}
	// main.go vive en backend/cmd/gen-abanderados-template: tres niveles arriba
	// está la raíz del repositorio.
	return filepath.Clean(filepath.Join(
		filepath.Dir(thisFile), "..", "..", "..",
		"frontend", "public", "plantilla-abanderados.xlsx",
	))
}
