import { CommonModule } from '@angular/common';
import { Component, DestroyRef, OnInit, computed, inject, signal } from '@angular/core';
import { takeUntilDestroyed } from '@angular/core/rxjs-interop';
import { EMPTY, Observable, catchError, map, of, switchMap } from 'rxjs';
import { IconComponent } from '../../shared/components/icon/icon.component';
import { Student } from '../../model/student';
import { StudentService } from '../../service/student.service';
import { SettingsService } from '../../service/settings.service';
import {
  GroupService,
  IdentifyResponse,
  MyGroupsResponse,
} from '../../service/group.service';
import { GroupView } from '../../model/group';
import { Invitation } from '../../model/invitations';

@Component({
  imports: [CommonModule, IconComponent],
  selector: 'app-groups',
  templateUrl: './groups.component.html',
})
export class GroupsComponent implements OnInit {
  private settingsService = inject(SettingsService);
  private groupService = inject(GroupService);
  private studentService = inject(StudentService);
  private readonly destroyRef = inject(DestroyRef);

  groupsEnabled = signal(false);
  identity = signal<Student | null>(null);
  loading = signal(false);
  error = signal('');
  notice = signal('');

  myGroup = signal<GroupView | null>(null);
  invitations = signal<Invitation[]>([]);
  sent = signal<Invitation[]>([]);

  // Listado público de grupos (siempre visible, con paginación propia).
  allGroups = signal<GroupView[]>([]);
  groupPage = signal(1);
  readonly groupPageSize = 12;

  // Identificación
  dni = signal('');
  email = signal('');

  // Creación de grupo
  showCreate = signal(false);
  solo = signal(false);
  students = signal<Student[]>([]);
  inviteQuery = signal('');
  invitees = signal<Student[]>([]);

  readonly identified = computed(() => !!this.identity());
  readonly inviteSearchDisabled = computed(() => this.invitees().length >= 2);
  readonly pending = computed(() =>
    this.invitations().filter((invitation) => invitation.status === 'PENDING')
  );
  readonly inviteMatches = computed(() => {
    const query = this.normalizeName(this.inviteQuery());
    if (query.length < 2) {
      return [];
    }
    const myID = this.identity()?.id;
    return this.students().filter(
      (student) =>
        this.normalizeName(student.name).includes(query) &&
        student.confirmed &&
        !student.groupID &&
        student.id !== myID
    );
  });

  readonly totalGroupPages = computed(() =>
    Math.max(Math.ceil(this.allGroups().length / this.groupPageSize), 1)
  );
  readonly currentGroupPage = computed(() =>
    Math.min(Math.max(this.groupPage(), 1), this.totalGroupPages())
  );
  readonly paginatedGroups = computed(() => {
    const start = (this.currentGroupPage() - 1) * this.groupPageSize;
    return this.allGroups().slice(start, start + this.groupPageSize);
  });

  ngOnInit(): void {
    this.loadSettings().pipe(takeUntilDestroyed(this.destroyRef)).subscribe();
    this.loadGroups();

    if (this.groupService.isIdentified()) {
      this.loadMyGroups();
    }
  }

  loadGroups(): void {
    this.groupService
      .getGroups()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (groups) => this.allGroups.set(groups || []),
        error: () => this.allGroups.set([]),
      });
  }

  changeGroupPage(direction: number): void {
    const next = this.currentGroupPage() + direction;
    if (next < 1 || next > this.totalGroupPages()) {
      return;
    }
    this.groupPage.set(next);
  }

  // Consulta el flag del admin. Ante cualquier error se mantiene cerrado
  // (fail-closed): nunca mostramos acciones que el backend rechazaría.
  private loadSettings(): Observable<boolean> {
    return this.settingsService.getSettings().pipe(
      map((settings) => {
        const enabled = settings.groupsEnabled === true;
        this.groupsEnabled.set(enabled);
        return enabled;
      }),
      catchError(() => {
        this.groupsEnabled.set(false);
        return of(false);
      })
    );
  }

  // Revalida el flag justo antes de cada acción: si el admin cerró la etapa
  // mientras la página estaba abierta, no se ejecuta la llamada.
  private withGroupsOpen<T>(action: () => Observable<T>): Observable<T> {
    return this.loadSettings().pipe(
      switchMap((enabled) => {
        if (!enabled) {
          this.loading.set(false);
          this.error.set('La formación de grupos fue cerrada por el administrador.');
          return EMPTY;
        }
        return action();
      })
    );
  }

  identify(event: Event): void {
    event.preventDefault();
    const dni = this.dni().trim();
    const email = this.email().trim();
    if (!dni || !email) {
      this.error.set('Ingresá tu DNI y tu correo electrónico.');
      return;
    }

    this.loading.set(true);
    this.error.set('');
    this.withGroupsOpen(() => this.groupService.identify(dni, email))
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (response: IdentifyResponse) => {
          this.identity.set(response.student);
          this.loading.set(false);
          this.loadMyGroups();
        },
        error: () => {
          this.error.set('No encontramos un ingresante confirmado con ese DNI y correo.');
          this.loading.set(false);
        },
      });
  }

  changeIdentity(): void {
    this.groupService.logout();
    this.identity.set(null);
    this.myGroup.set(null);
    this.invitations.set([]);
    this.sent.set([]);
    this.dni.set('');
    this.email.set('');
    this.showCreate.set(false);
    this.invitees.set([]);
    this.notice.set('');
    this.error.set('');
  }

  loadMyGroups(): void {
    this.groupService
      .getMyGroups()
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: (response: MyGroupsResponse) => {
          this.myGroup.set(response.group);
          this.invitations.set(response.invitations || []);
          this.sent.set(response.sentInvitations || []);
        },
        error: (err) => {
          // Solo un 401 significa identificación vencida. Otros errores (CORS,
          // red, 500) no deben borrar la identidad ni hacer "refrescar" la vista.
          if (err?.status === 401) {
            this.changeIdentity();
          } else {
            this.error.set('No pudimos cargar tus grupos. Intentá nuevamente.');
          }
        },
      });
  }

  openCreate(): void {
    this.error.set('');
    this.solo.set(false);
    this.inviteQuery.set('');
    this.invitees.set([]);
    this.showCreate.set(true);
    if (this.students().length === 0) {
      this.studentService
        .getStudents()
        .pipe(takeUntilDestroyed(this.destroyRef))
        .subscribe({
          next: (data) => this.students.set(data),
          error: () => this.error.set('No pudimos cargar el listado de ingresantes.'),
        });
    }
  }

  closeCreate(): void {
    this.showCreate.set(false);
    this.error.set('');
  }

  toggleInvitee(student: Student): void {
    const current = this.invitees();
    if (current.some((item) => item.id === student.id)) {
      this.invitees.set(current.filter((item) => item.id !== student.id));
      return;
    }
    if (current.length >= 2) {
      this.error.set('Podés invitar como máximo a 2 ingresantes (3 con vos).');
      return;
    }
    this.error.set('');
    this.invitees.set([...current, student]);
    // Los resultados flotantes se derivan de este signal: limpiarlo oculta la
    // lista y deja el buscador listo para la próxima búsqueda.
    this.inviteQuery.set('');
  }

  isInvitee(student: Student): boolean {
    return this.invitees().some((item) => item.id === student.id);
  }

  // Búsqueda por nombre tolerando tildes y mayúsculas.
  private normalizeName(value: string): string {
    return value
      .toLocaleLowerCase('es')
      .normalize('NFD')
      .replace(/[\u0300-\u036f]/g, '')
      .trim();
  }

  createGroup(): void {
    const invitees = this.solo() ? [] : this.invitees().map((student) => student.id);
    if (!this.solo() && invitees.length === 0) {
      this.error.set('Elegí al menos un ingresante o activá "En solitario".');
      return;
    }

    this.loading.set(true);
    this.error.set('');
    this.withGroupsOpen(() => this.groupService.createGroup(this.solo(), invitees))
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.loading.set(false);
          this.showCreate.set(false);
          this.notice.set('Grupo creado correctamente.');
          this.loadMyGroups();
          this.loadGroups();
        },
        error: (err) => {
          this.loading.set(false);
          this.error.set(err?.error || 'No se pudo crear el grupo.');
        },
      });
  }

  accept(invitation: Invitation): void {
    this.error.set('');
    this.withGroupsOpen(() => this.groupService.acceptInvitation(invitation.id))
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.notice.set('Te incorporaste al grupo.');
          this.loadMyGroups();
          this.loadGroups();
        },
        error: (err) => this.error.set(err?.error || 'No se pudo aceptar la invitación.'),
      });
  }

  reject(invitation: Invitation): void {
    this.error.set('');
    this.withGroupsOpen(() => this.groupService.rejectInvitation(invitation.id))
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.notice.set('Invitación rechazada. Ya podés armar tu grupo.');
          this.loadMyGroups();
        },
        error: (err) => this.error.set(err?.error || 'No se pudo rechazar la invitación.'),
      });
  }

  cancel(invitation: Invitation): void {
    this.error.set('');
    this.withGroupsOpen(() => this.groupService.cancelInvitation(invitation.id))
      .pipe(takeUntilDestroyed(this.destroyRef))
      .subscribe({
        next: () => {
          this.notice.set('Invitación cancelada.');
          this.loadMyGroups();
        },
        error: (err) => this.error.set(err?.error || 'No se pudo cancelar la invitación.'),
      });
  }

  statusLabel(status: string): string {
    switch (status) {
      case 'PENDING':
        return 'Pendiente';
      case 'ACCEPTED':
        return 'Aceptada';
      case 'REJECTED':
        return 'Rechazada';
      case 'CANCELED':
        return 'Cancelada';
      default:
        return status;
    }
  }

  dismissNotice(): void {
    this.notice.set('');
  }
}
