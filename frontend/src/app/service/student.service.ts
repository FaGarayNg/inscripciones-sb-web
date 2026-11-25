import { HttpClient } from '@angular/common/http';
import { inject, Service } from '@angular/core';
import { Observable } from 'rxjs';
import { Student } from '../model/student';
import { API_URL } from '../core/config/api-url.token';

// Respuesta del endpoint de importación
export interface ImportResult {
  success: number;
  errors: string[];
}

@Service()
export class StudentService {
    private apiUrl = inject(API_URL);
    private httpClient = inject(HttpClient);

    // Obtener todos los alumnos (público: es la lista que consulta la comunidad).
    // La API pública no expone el correo electrónico.
    getStudents(): Observable<Student[]> {
        return this.httpClient.get<Student[]>(`${this.apiUrl}/students`);
    }

    // Listado completo para el panel admin (requiere token; incluye email).
    getAdminStudents(): Observable<Student[]> {
        return this.httpClient.get<Student[]>(`${this.apiUrl}/admin/students`);
    }

    // Crear un nuevo alumno (requiere token de administrador)
    createStudent(student: Student): Observable<Student> {
        return this.httpClient.post<Student>(`${this.apiUrl}/admin/students`, student);
    }

    // Actualizar un alumno existente (requiere token de administrador)
    updateStudent(id: string, student: Student): Observable<Student> {
        return this.httpClient.put<Student>(`${this.apiUrl}/admin/students/${id}`, student);
    }

    // Confirmar el banco de un alumno (flujo público del ingresante).
    // El DNI se envía como segunda verificación: el ID del alumno es público.
    confirmStudent(id: string, email: string, dni: string): Observable<Student> {
        return this.httpClient.post<Student>(`${this.apiUrl}/students/${id}/confirm`, { email, dni });
    }

    // Exportar el listado ordenado por grupos en un .xlsx (requiere token).
    exportStudents(): Observable<Blob> {
        return this.httpClient.get(`${this.apiUrl}/admin/students/export`, { responseType: 'blob' });
    }

    // Eliminar un alumno (requiere token de administrador)
    deleteStudent(id: string): Observable<void> {
        return this.httpClient.delete<void>(`${this.apiUrl}/admin/students/${id}`);
    }

    // Importar alumnos desde Excel
    // La escuela se toma de la columna E del archivo; si falta, queda secundaria
    importStudents(file: File): Observable<ImportResult> {
        const formData = new FormData();
        formData.append('file', file);
        return this.httpClient.post<ImportResult>(`${this.apiUrl}/admin/students/import`, formData);
    }

}
