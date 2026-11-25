export type InvitationStatus = 'PENDING' | 'ACCEPTED' | 'REJECTED' | 'CANCELED';

// Invitación enviada por un grupo a un ingresante (colección "invitations").
export interface Invitation {
  id: string;
  groupId: string;
  studentId: string;
  studentName: string;
  leaderName: string;
  status: InvitationStatus;
  createdAt: string;
}
