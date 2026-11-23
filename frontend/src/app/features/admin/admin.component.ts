import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { AuthService } from '../../service/auth.service';
import { StudentService, ImportResult } from '../../service/student.service';
import { AbanderadoService, AbanderadoImportResult } from '../../service/abanderado.service';
import { Abanderado } from '../../model/abanderado';
import { GroupService } from '../../service/group.service';
import { SettingsService } from '../../service/settings.service';
import { Student } from '../../model/student';
import { Group } from '../../model/group';
import { IconComponent } from '../../shared/components/icon/icon.component';

type AdminSection = 'resumen' | 'alumnos' | 'grupos';

interface AdminStudent {
  id: string;
  dni: string;
  name: string;
  email: string;
  instance: string;
  title: string;
  school: 'Secundaria' | 'Deportiva';
  confirmed: boolean;
}

interface AdminGroup {
  id: string;
  name: string;
  mode: 'Grupal' | 'En solitario';
  members: string[];
}

@Component({
  selector: 'app-admin',
  standalone: true,
  imports: [CommonModule, FormsModule, IconComponent],
  templateUrl: './admin.component.html',
  styleUrl: './admin.component.css'
})
export class AdminComponent implements OnInit {
  section = signal<AdminSection>('resumen');
  students = signal<AdminStudent[]>([]);
  groups = signal<AdminGroup[]>([]);
  loading = signal(true);
  studentQuery = signal('');
  // Paginación de la tabla de alumnos
  readonly pageSize = 10;
  currentPage = signal(1);
  editingStudent = signal<AdminStudent | null>(null);
  isNewStudent = signal(false);
  notice = signal('');
  groupsEnabled = signal(false);
  savingGroups = signal(false);

  // Importación desde Excel
  importResult = signal<ImportResult | null>(null);
  importing = signal(false);

  // Lista de abanderados
  abanderados = signal<Abanderado[]>([]);
  abanderadoImportResult = signal<AbanderadoImportResult | null>(null);
  abanderadoImporting = signal(false);

  constructor(
    private authService: AuthService,
    private studentService: StudentService,
    private abanderadoService: AbanderadoService,
    private groupService: GroupService,
    private settingsService: SettingsService,
    private router: Router
  ) {}

  ngOnInit(): void {
    this.loadData();
  }

  loadData(): void {
    this.loading.set(true);

    this.studentService.getStudents().subscribe({
      next: (data) => {
        const mapped = data.map(s => ({
          id: s.id || '',
          dni: s.dni,
          name: s.name,
          email: s.email,
          instance: s.instance,
          title: s.title,
          school: s.school === 'deportiva' ? ('Deportiva' as const) : ('Secundaria' as const),
          confirmed: s.confirmed
        }));
        this.students.set(mapped);
        this.loading.set(false);
      },
      error: () => {
        this.loading.set(false);
        this.showNotice('Error al cargar alumnos');
      }
    });

    this.abanderadoService.getAbanderados().subscribe({
      next: (data) => this.abanderados.set(data),
      error: () => this.showNotice('Error al cargar abanderados')
    });

    this.groupService.getGroups().subscribe({
      next: (data) => {
        const mapped = data.map(g => ({
          id: g.id || '',
          name: `Grupo de ${g.leader?.name || 'Sin nombre'}`,
          mode: (g.members?.length || 0) > 0 ? 'Grupal' as const : 'En solitario' as const,
          members: (g.members || []).map(m => m.name)
        }));
        this.groups.set(mapped);
      },
      error: () => {
        this.showNotice('Error al cargar grupos');
      }
    });

    // Disponibilidad de grupos: la controla el secretario desde este panel
    this.settingsService.getSettings().subscribe({
      next: (settings) => this.groupsEnabled.set(settings.groupsEnabled === true),
      error: () => this.groupsEnabled.set(false)
    });
  }

  setSection(section: AdminSection): void {
    this.section.set(section);
  }

  // Al cambiar la búsqueda volvemos a la primera página
  onStudentSearch(value: string): void {
    this.studentQuery.set(value);
    this.currentPage.set(1);
  }

  showNotice(message: string): void {
    this.notice.set(message);
    setTimeout(() => this.notice.set(''), 3200);
  }

  // Alumnos
  startNewStudent(): void {
    this.editingStudent.set({
      id: '',
      dni: '',
      name: '',
      email: '',
      instance: 'Primera instancia',
      title: '',
      school: 'Secundaria',
      confirmed: false
    });
    this.isNewStudent.set(true);
  }

  editStudent(student: AdminStudent): void {
    this.editingStudent.set({ ...student });
    this.isNewStudent.set(false);
  }

  saveStudent(event: Event): void {
    event.preventDefault();
    const student = this.editingStudent();
    if (!student?.name || !student.dni) return;

    const studentData: Student = {
      id: student.id,
      dni: student.dni,
      name: student.name,
      email: student.email,
      instance: student.instance,
      title: student.title,
      school: student.school === 'Deportiva' ? 'deportiva' : 'secundaria',
      hasGroup: false,
      alone: false,
      confirmed: student.confirmed
    };

    if (this.isNewStudent()) {
      this.studentService.createStudent(studentData).subscribe({
        next: () => {
          this.loadData();
          this.showNotice('Alumno agregado correctamente.');
          this.editingStudent.set(null);
        },
        error: () => this.showNotice('Error al agregar alumno')
      });
    } else {
      this.studentService.updateStudent(student.id, studentData).subscribe({
        next: () => {
          this.loadData();
          this.showNotice('Datos del alumno actualizados.');
          this.editingStudent.set(null);
        },
        error: () => this.showNotice('Error al actualizar alumno')
      });
    }
  }

  deleteStudent(id: string): void {
    if (!confirm('¿Eliminar este alumno?')) return;
    this.studentService.deleteStudent(id).subscribe({
      next: () => {
        this.students.update(s => s.filter(st => st.id !== id));
        this.showNotice('Alumno eliminado del sistema.');
      },
      error: () => this.showNotice('Error al eliminar alumno')
    });
  }

  importStudents(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    if (!file.name.toLowerCase().endsWith('.xlsx')) {
      this.showNotice('Solo se permiten archivos .xlsx');
      input.value = '';
      return;
    }

    this.importing.set(true);
    this.importResult.set(null);

    this.studentService.importStudents(file).subscribe({
      next: (result) => {
        this.importResult.set({
          success: result.success || 0,
          errors: result.errors || [],
        });
        this.importing.set(false);
        this.loadData();
      },
      error: () => {
        this.importResult.set({
          success: 0,
          errors: ['No se pudo procesar el archivo. Revisá que sea un .xlsx válido.'],
        });
        this.importing.set(false);
      },
    });

    // Permite reimportar el mismo archivo más adelante
    input.value = '';
  }

  // Cuántos errores mostrar antes de resumir el resto
  get visibleImportErrors(): string[] {
    return (this.importResult()?.errors || []).slice(0, 8);
  }

  get hiddenImportErrors(): number {
    const total = this.importResult()?.errors.length || 0;
    return Math.max(total - this.visibleImportErrors.length, 0);
  }

  dismissImportResult(): void {
    this.importResult.set(null);
  }

  importAbanderados(event: Event): void {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;

    if (!file.name.toLowerCase().endsWith('.xlsx')) {
      this.showNotice('Solo se permiten archivos .xlsx');
      input.value = '';
      return;
    }

    this.abanderadoImporting.set(true);
    this.abanderadoImportResult.set(null);

    this.abanderadoService.importAbanderados(file).subscribe({
      next: (result) => {
        this.abanderadoImportResult.set({
          success: result.success || 0,
          errors: result.errors || [],
        });
        this.abanderadoImporting.set(false);
        this.loadData();
      },
      error: () => {
        this.abanderadoImportResult.set({
          success: 0,
          errors: ['No se pudo procesar el archivo. Revisá que sea un .xlsx válido.'],
        });
        this.abanderadoImporting.set(false);
      },
    });

    // Permite reimportar el mismo archivo más adelante
    input.value = '';
  }

  deleteAbanderado(id: string): void {
    const abanderado = this.abanderados().find(a => a.id === id);
    if (!confirm(`¿Quitar a ${abanderado?.name || 'este abanderado'} de la lista?`)) return;

    this.abanderadoService.deleteAbanderado(id).subscribe({
      next: () => {
        this.abanderados.update(list => list.filter(a => a.id !== id));
        this.showNotice('Abanderado quitado de la lista.');
      },
      error: () => this.showNotice('Error al quitar abanderado')
    });
  }

  // Cuántos errores mostrar antes de resumir el resto
  get visibleAbanderadoImportErrors(): string[] {
    return (this.abanderadoImportResult()?.errors || []).slice(0, 8);
  }

  get hiddenAbanderadoImportErrors(): number {
    const total = this.abanderadoImportResult()?.errors.length || 0;
    return Math.max(total - this.visibleAbanderadoImportErrors.length, 0);
  }

  dismissAbanderadoImportResult(): void {
    this.abanderadoImportResult.set(null);
  }

  // Grupos
  removeMember(groupId: string, member: string): void {
    if (!confirm(`¿Quitar a ${member} del grupo?`)) return;

    this.groupService.getGroups().subscribe({
      next: (groups) => {
        const group = groups.find(g => g.id === groupId);
        const memberObj = group?.members?.find(m => m.name === member);
        if (memberObj?.id) {
          this.groupService.removeMember(groupId, memberObj.id).subscribe({
            next: () => {
              this.loadData();
              this.showNotice('Integrante removido del grupo.');
            },
            error: () => this.showNotice('Error al remover integrante')
          });
        }
      }
    });
  }

  deleteAdminGroup(groupId: string): void {
    if (!confirm('¿Eliminar este grupo?')) return;
    this.groupService.deleteGroup(groupId).subscribe({
      next: () => {
        this.groups.update(g => g.filter(gr => gr.id !== groupId));
        this.showNotice('Grupo eliminado correctamente.');
      },
      error: () => this.showNotice('Error al eliminar grupo')
    });
  }

  // Disponibilidad de grupos
  toggleGroupsEnabled(): void {
    if (this.savingGroups()) return;
    const desired = !this.groupsEnabled();

    this.savingGroups.set(true);
    this.settingsService.updateSettings({ groupsEnabled: desired }).subscribe({
      next: (settings) => {
        this.groupsEnabled.set(settings.groupsEnabled);
        this.savingGroups.set(false);
        this.showNotice(
          settings.groupsEnabled
            ? 'Formación de grupos habilitada.'
            : 'Formación de grupos deshabilitada.'
        );
      },
      error: () => {
        this.savingGroups.set(false);
        this.showNotice('No se pudo actualizar la disponibilidad de grupos.');
      }
    });
  }

  // Utilidades
  get confirmedCount(): number {
    return this.students().filter(s => s.confirmed).length;
  }

  get groupedStudentCount(): number {
    return new Set(this.groups().flatMap(g => g.members)).size;
  }

  get maxStudents(): number {
    return Math.max(this.students().length, 1);
  }

  get filteredStudents(): AdminStudent[] {
    const q = this.studentQuery().toLowerCase();
    if (!q) return this.students();
    return this.students().filter(s =>
      `${s.name} ${s.dni}`.toLowerCase().includes(q)
    );
  }

  get totalStudentPages(): number {
    return Math.max(Math.ceil(this.filteredStudents.length / this.pageSize), 1);
  }

  // La página se limita al rango vigente: si se borran alumnos y la lista
  // se achica, no quedamos mostrando una página que ya no existe.
  get studentPage(): number {
    return Math.min(Math.max(this.currentPage(), 1), this.totalStudentPages);
  }

  get paginatedStudents(): AdminStudent[] {
    const start = (this.studentPage - 1) * this.pageSize;
    return this.filteredStudents.slice(start, start + this.pageSize);
  }

  get studentRangeStart(): number {
    return this.filteredStudents.length === 0 ? 0 : (this.studentPage - 1) * this.pageSize + 1;
  }

  get studentRangeEnd(): number {
    return Math.min(this.studentPage * this.pageSize, this.filteredStudents.length);
  }

  changeStudentPage(direction: number): void {
    const next = this.studentPage + direction;
    if (next < 1 || next > this.totalStudentPages) return;
    this.currentPage.set(next);
  }

  updateStudentField(field: keyof AdminStudent, value: any): void {
    const current = this.editingStudent();
    if (current) {
      this.editingStudent.set({ ...current, [field]: value });
    }
  }

  logout(): void {
    this.authService.logout();
    this.router.navigate(['/inicio']);
  }
}
