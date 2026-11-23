import { Component } from '@angular/core';
import { StudentListComponent } from './student-list/student-list.component';
import { AbanderadosListComponent } from './abanderados-list/abanderados-list.component';
import { IconComponent } from '../../shared/components/icon/icon.component';

@Component({
  selector: 'app-home',
  standalone: true,
  imports: [StudentListComponent, AbanderadosListComponent, IconComponent],
  templateUrl: './home.component.html'
})
export class HomeComponent {
}
