import { HttpClient, HttpHeaders } from '@angular/common/http';
import { inject, Service } from '@angular/core';
import { Observable, tap } from 'rxjs';
import { API_URL } from '../core/config/api-url.token';
import { GroupView } from '../model/group';
import { Invitation } from '../model/invitations';
import { Student } from '../model/student';

export interface IdentifyResponse {
  token: string;
  student: Student;
}

export interface MyGroupsResponse {
  group: GroupView | null;
  invitations: Invitation[];
  sentInvitations: Invitation[];
}

export interface CreateGroupResponse {
  group: GroupView;
  invitations: Invitation[];
}

export interface MessageResponse {
  message: string;
}

@Service()
export class GroupService {
  private apiUrl = inject(API_URL);
  private httpClient = inject(HttpClient);
  private readonly tokenKey = 'group_token';

  // Identidad del ingresante: se pide DNI + email y se guarda un token de
  // alumno en sessionStorage (distinto del JWT de administrador).
  identify(dni: string, email: string): Observable<IdentifyResponse> {
    return this.httpClient
      .post<IdentifyResponse>(`${this.apiUrl}/groups/identify`, { dni, email })
      .pipe(tap((response) => sessionStorage.setItem(this.tokenKey, response.token)));
  }

  logout(): void {
    sessionStorage.removeItem(this.tokenKey);
  }

  getToken(): string | null {
    return sessionStorage.getItem(this.tokenKey);
  }

  isIdentified(): boolean {
    return !!this.getToken();
  }

  private studentHeaders(): HttpHeaders {
    return new HttpHeaders({ 'X-Student-Token': this.getToken() || '' });
  }

  // Listado público de grupos (con nombres, sin correos).
  getGroups(): Observable<GroupView[]> {
    return this.httpClient.get<GroupView[]>(`${this.apiUrl}/groups`);
  }

  getMyGroups(): Observable<MyGroupsResponse> {
    return this.httpClient.get<MyGroupsResponse>(`${this.apiUrl}/groups/me`, {
      headers: this.studentHeaders(),
    });
  }

  // `invitees` son IDs de alumno (la búsqueda en la UI es por nombre).
  createGroup(solo: boolean, invitees: string[]): Observable<CreateGroupResponse> {
    return this.httpClient.post<CreateGroupResponse>(
      `${this.apiUrl}/groups`,
      { solo, invitees },
      { headers: this.studentHeaders() }
    );
  }

  acceptInvitation(id: string): Observable<GroupView> {
    return this.httpClient.post<GroupView>(
      `${this.apiUrl}/groups/invitations/${id}/accept`,
      {},
      { headers: this.studentHeaders() }
    );
  }

  rejectInvitation(id: string): Observable<MessageResponse> {
    return this.httpClient.post<MessageResponse>(
      `${this.apiUrl}/groups/invitations/${id}/reject`,
      {},
      { headers: this.studentHeaders() }
    );
  }

  cancelInvitation(id: string): Observable<MessageResponse> {
    return this.httpClient.post<MessageResponse>(
      `${this.apiUrl}/groups/invitations/${id}/cancel`,
      {},
      { headers: this.studentHeaders() }
    );
  }

  // Panel administrativo (requiere el JWT de admin).
  getAdminGroups(): Observable<GroupView[]> {
    return this.httpClient.get<GroupView[]>(`${this.apiUrl}/admin/groups`);
  }

  deleteGroup(id: string): Observable<void> {
    return this.httpClient.delete<void>(`${this.apiUrl}/admin/groups/${id}`);
  }

  removeMember(groupId: string, memberId: string): Observable<GroupView> {
    return this.httpClient.delete<GroupView>(
      `${this.apiUrl}/admin/groups/${groupId}/members/${memberId}`
    );
  }
}
