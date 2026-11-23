import { Component, OnInit, signal } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Abanderado } from '../../../model/abanderado';
import { AbanderadoService } from '../../../service/abanderado.service';
import { IconComponent } from '../../../shared/components/icon/icon.component';

// Listado público de abanderados. No lleva filtros: solo muestra la lista
// cargada desde el panel administrativo, con la misma tabla que los ingresantes.
@Component({
  selector: 'app-abanderados-list',
  standalone: true,
  imports: [CommonModule, IconComponent],
  templateUrl: './abanderados-list.component.html',
})
export class AbanderadosListComponent implements OnInit {
  abanderados = signal<Abanderado[]>([]);
  currentPage = signal(1);
  readonly itemsPerPage = 5;

  constructor(private abanderadoService: AbanderadoService) {}

  ngOnInit(): void {
    this.abanderadoService.getAbanderados().subscribe({
      next: (data) => this.abanderados.set(data),
      error: () => this.abanderados.set([]),
    });
  }

  get totalPages(): number {
    return Math.max(Math.ceil(this.abanderados().length / this.itemsPerPage), 1);
  }

  get page(): number {
    return Math.min(Math.max(this.currentPage(), 1), this.totalPages);
  }

  get paginated(): Abanderado[] {
    const start = (this.page - 1) * this.itemsPerPage;
    return this.abanderados().slice(start, start + this.itemsPerPage);
  }

  changePage(direction: number): void {
    const next = this.page + direction;
    if (next < 1 || next > this.totalPages) return;
    this.currentPage.set(next);
  }
}
