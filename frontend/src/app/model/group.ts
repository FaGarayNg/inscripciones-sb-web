import { Student } from './student';

// Grupo persistido: guarda referencias por ID, no copias de los alumnos.
export interface Group {
  id: string;
  leaderId: string;
  memberIds: string[];
  solo: boolean;
  full?: boolean;
  createdAt: string;
}

// Vista pública del grupo con los alumnos ya resueltos por el backend.
export interface GroupView {
  id: string;
  leader: Student;
  members: Student[];
  solo: boolean;
  full: boolean;
  createdAt: string;
}
