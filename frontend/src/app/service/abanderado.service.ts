import { HttpClient } from '@angular/common/http';
import { inject, Service } from '@angular/core';
import { Observable } from 'rxjs';
import { Abanderado } from '../model/abanderado';
import { API_URL } from '../core/config/api-url.token';

// Respuesta del endpoint de importación de abanderados
export interface AbanderadoImportResult {
  success: number;
  errors: string[];
}

@Service()
export class AbanderadoService {
  private apiUrl = inject(API_URL);
  private httpClient = inject(HttpClient);

  // Lista completa de abanderados (pública)
  getAbanderados(): Observable<Abanderado[]> {
    return this.httpClient.get<Abanderado[]>(`${this.apiUrl}/abanderados`);
  }

  // Importar la lista desde Excel (requiere token de administrador)
  importAbanderados(file: File): Observable<AbanderadoImportResult> {
    const formData = new FormData();
    formData.append('file', file);
    return this.httpClient.post<AbanderadoImportResult>(
      `${this.apiUrl}/admin/abanderados/import`,
      formData
    );
  }

  // Quitar un abanderado de la lista (requiere token de administrador)
  deleteAbanderado(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.apiUrl}/admin/abanderados/${id}`);
  }
}
