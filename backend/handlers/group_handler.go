package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"servidor-angular/config"
	"servidor-angular/models"
	"servidor-angular/repository"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// GroupHandler handles HTTP requests for groups and their invitations.
type GroupHandler struct {
	groupRepo      *repository.GroupRepository
	studentRepo    *repository.StudentRepository
	invitationRepo *repository.InvitationRepository
	settingRepo    *repository.SettingRepository
}

// NewGroupHandler creates a new group handler
func NewGroupHandler() *GroupHandler {
	return &GroupHandler{
		groupRepo:      repository.NewGroupRepository(),
		studentRepo:    repository.NewStudentRepository(),
		invitationRepo: repository.NewInvitationRepository(),
		settingRepo:    repository.NewSettingRepository(),
	}
}

// groupsOpen refleja el flag que el secretario togglea desde el panel. La UI
// lo consulta, pero el backend también lo exige para crear/aceptar/rechazar.
func (h *GroupHandler) groupsOpen() bool {
	settings, err := h.settingRepo.Get()
	return err == nil && settings.GroupsEnabled
}

// publicGroups arma las vistas con nombres, consultando los alumnos por ID.
func (h *GroupHandler) publicGroups(groups []models.Group) ([]models.PublicGroup, error) {
	ids := make([]primitive.ObjectID, 0, len(groups))
	for _, group := range groups {
		ids = append(ids, group.LeaderID)
		ids = append(ids, group.MemberIDs...)
	}
	students, err := h.studentRepo.FindByIDs(ids)
	if err != nil {
		return nil, err
	}

	views := make([]models.PublicGroup, 0, len(groups))
	for _, group := range groups {
		views = append(views, models.NewPublicGroup(group, students))
	}
	return views, nil
}

// GetAllGroups handles GET /api/groups (público, sin correos).
func (h *GroupHandler) GetAllGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.groupRepo.FindAll()
	if err != nil {
		http.Error(w, "Error al obtener grupos", http.StatusInternalServerError)
		return
	}
	views, err := h.publicGroups(groups)
	if err != nil {
		http.Error(w, "Error al obtener los alumnos de los grupos", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, views)
}

// GetGroup handles GET /api/groups/{id} (público).
func (h *GroupHandler) GetGroup(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	group, err := h.groupRepo.FindByID(id)
	if err != nil {
		http.Error(w, "Grupo no encontrado", http.StatusNotFound)
		return
	}
	views, err := h.publicGroups([]models.Group{*group})
	if err != nil || len(views) == 0 {
		http.Error(w, "Error al obtener el grupo", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, views[0])
}

// GetAllGroupsAdmin handles GET /api/admin/groups (protegido).
func (h *GroupHandler) GetAllGroupsAdmin(w http.ResponseWriter, r *http.Request) {
	groups, err := h.groupRepo.FindAll()
	if err != nil {
		http.Error(w, "Error al obtener grupos", http.StatusInternalServerError)
		return
	}
	views, err := h.publicGroups(groups)
	if err != nil {
		http.Error(w, "Error al obtener los alumnos de los grupos", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, views)
}

// IdentifyRequest son las credenciales del ingresante en el flujo de grupos.
type IdentifyRequest struct {
	DNI   string `json:"dni"`
	Email string `json:"email"`
}

// Identify handles POST /api/groups/identify (público). Valida DNI + email
// contra un alumno que ya confirmó su banco y devuelve un token de alumno.
func (h *GroupHandler) Identify(w http.ResponseWriter, r *http.Request) {
	if !h.groupsOpen() {
		http.Error(w, "La formación de grupos está cerrada", http.StatusForbidden)
		return
	}

	var req IdentifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Error en los datos enviados", http.StatusBadRequest)
		return
	}
	dni := strings.TrimSpace(req.DNI)
	email := strings.TrimSpace(req.Email)
	if dni == "" || email == "" {
		http.Error(w, "Ingresá tu DNI y tu correo electrónico", http.StatusBadRequest)
		return
	}

	student, err := h.studentRepo.FindByDNIAndEmail(dni, email)
	if err != nil {
		http.Error(w, "No encontramos un ingresante confirmado con ese DNI y correo", http.StatusUnauthorized)
		return
	}
	token, err := config.GenerateStudentToken(student.ID.Hex())
	if err != nil {
		http.Error(w, "Error al generar la identificación", http.StatusInternalServerError)
		return
	}

	jsonResponse(w, map[string]interface{}{
		"token":   token,
		"student": models.ToPublicStudent(*student),
	})
}

// MyGroupsResponse es el estado del alumno en la sección Grupos.
type MyGroupsResponse struct {
	Group       *models.PublicGroup `json:"group"`
	Invitations []models.Invitation `json:"invitations"`
	Sent        []models.Invitation `json:"sentInvitations"`
}

// GetMyGroups handles GET /api/groups/me (token de alumno).
func (h *GroupHandler) GetMyGroups(w http.ResponseWriter, r *http.Request) {
	studentID, err := StudentIDFromRequest(r)
	if err != nil {
		http.Error(w, "Identificación inválida", http.StatusUnauthorized)
		return
	}

	response := MyGroupsResponse{
		Invitations: []models.Invitation{},
		Sent:        []models.Invitation{},
	}

	student, err := h.studentRepo.FindByID(studentID)
	if err != nil {
		http.Error(w, "Alumno no encontrado", http.StatusNotFound)
		return
	}

	if student.GroupID != "" {
		groupID, err := primitive.ObjectIDFromHex(student.GroupID)
		if err == nil {
			if group, err := h.groupRepo.FindByID(groupID); err == nil {
				views, err := h.publicGroups([]models.Group{*group})
				if err == nil && len(views) > 0 {
					response.Group = &views[0]
				}
				sent, err := h.invitationRepo.FindByGroup(groupID)
				if err == nil {
					response.Sent = sent
				}
			}
		}
	}

	invitations, err := h.invitationRepo.FindByStudent(studentID)
	if err != nil {
		http.Error(w, "Error al obtener invitaciones", http.StatusInternalServerError)
		return
	}
	response.Invitations = invitations

	jsonResponse(w, response)
}

// CreateRequest define cómo se crea un grupo. En modo solitario se ignoran
// los invitados; si no, se aceptan entre 1 y 2 IDs de alumno. Se usa el ID y
// no el nombre porque varios ingresantes pueden llamarse igual.
type CreateRequest struct {
	Solo     bool     `json:"solo"`
	Invitees []string `json:"invitees"`
}

// Create handles POST /api/groups (token de alumno).
func (h *GroupHandler) Create(w http.ResponseWriter, r *http.Request) {
	if !h.groupsOpen() {
		http.Error(w, "La formación de grupos está cerrada", http.StatusForbidden)
		return
	}
	studentID, err := StudentIDFromRequest(r)
	if err != nil {
		http.Error(w, "Identificación inválida", http.StatusUnauthorized)
		return
	}
	leader, err := h.studentRepo.FindByID(studentID)
	if err != nil || !leader.Confirmed {
		http.Error(w, "Alumno no encontrado o sin banco confirmado", http.StatusBadRequest)
		return
	}
	if leader.GroupID != "" {
		http.Error(w, "Ya pertenecés a un grupo", http.StatusConflict)
		return
	}

	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Error en los datos enviados", http.StatusBadRequest)
		return
	}

	invitees := make([]models.Student, 0, 2)
	if !req.Solo {
		if len(req.Invitees) < 1 || len(req.Invitees) > 2 {
			http.Error(w, "Elegí entre 1 y 2 ingresantes para invitar", http.StatusBadRequest)
			return
		}
		seen := make(map[primitive.ObjectID]bool)
		for _, raw := range req.Invitees {
			inviteeID, err := primitive.ObjectIDFromHex(strings.TrimSpace(raw))
			if err != nil {
				http.Error(w, "ID de ingresante inválido", http.StatusBadRequest)
				return
			}
			if seen[inviteeID] || inviteeID == studentID {
				http.Error(w, "Los invitados deben ser distintos y no podés invitarte a vos mismo", http.StatusBadRequest)
				return
			}
			seen[inviteeID] = true

			invitee, err := h.studentRepo.FindByID(inviteeID)
			if err != nil || !invitee.Confirmed || invitee.GroupID != "" {
				http.Error(w, "Uno de los ingresantes ya no está disponible", http.StatusBadRequest)
				return
			}
			invitees = append(invitees, *invitee)
		}
	}

	group := models.Group{
		LeaderID:  studentID,
		MemberIDs: []primitive.ObjectID{},
		Solo:      req.Solo,
		CreatedAt: time.Now().UTC(),
	}
	if err := h.groupRepo.Create(&group); err != nil {
		http.Error(w, "Error al crear el grupo", http.StatusInternalServerError)
		return
	}

	invitations := make([]models.Invitation, 0, len(invitees))
	for _, invitee := range invitees {
		invitations = append(invitations, models.Invitation{
			GroupID:     group.ID,
			StudentID:   invitee.ID,
			StudentName: invitee.Name,
			LeaderName:  leader.Name,
			Status:      models.InvitationPending,
			CreatedAt:   time.Now().UTC(),
		})
	}
	if err := h.invitationRepo.CreateMany(invitations); err != nil {
		_ = h.groupRepo.Delete(group.ID)
		http.Error(w, "Error al enviar las invitaciones", http.StatusInternalServerError)
		return
	}
	if err := h.studentRepo.SetGroupMembership(studentID, group.ID, req.Solo); err != nil {
		_ = h.invitationRepo.CancelByGroup(group.ID)
		_ = h.groupRepo.Delete(group.ID)
		http.Error(w, "Error al asociarte al grupo", http.StatusInternalServerError)
		return
	}

	views, err := h.publicGroups([]models.Group{group})
	if err != nil || len(views) == 0 {
		http.Error(w, "Error al obtener el grupo creado", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]interface{}{
		"group":       views[0],
		"invitations": invitations,
	})
}

// Accept handles POST /api/groups/invitations/{id}/accept (token de alumno).
func (h *GroupHandler) Accept(w http.ResponseWriter, r *http.Request) {
	if !h.groupsOpen() {
		http.Error(w, "La formación de grupos está cerrada", http.StatusForbidden)
		return
	}
	studentID, err := StudentIDFromRequest(r)
	if err != nil {
		http.Error(w, "Identificación inválida", http.StatusUnauthorized)
		return
	}
	invitationID, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}

	invitation, err := h.invitationRepo.FindPendingByID(invitationID)
	if err != nil {
		http.Error(w, "Invitación no encontrada o ya respondida", http.StatusNotFound)
		return
	}
	if invitation.StudentID != studentID {
		http.Error(w, "Esta invitación no es para vos", http.StatusForbidden)
		return
	}
	student, err := h.studentRepo.FindByID(studentID)
	if err != nil || student.GroupID != "" {
		http.Error(w, "Ya pertenecés a un grupo", http.StatusConflict)
		return
	}

	// El filtro del repo garantiza cupo y grupo no solitario de forma atómica.
	group, err := h.groupRepo.AddMember(invitation.GroupID, studentID)
	if err != nil {
		http.Error(w, "El grupo ya está completo o ya no está disponible", http.StatusConflict)
		return
	}
	if err := h.invitationRepo.SetStatus(invitationID, models.InvitationAccepted); err != nil {
		_ = h.groupRepo.RemoveMember(invitation.GroupID, studentID)
		http.Error(w, "Error al aceptar la invitación", http.StatusInternalServerError)
		return
	}
	if _, err := h.invitationRepo.CancelPendingForStudent(studentID, invitation.GroupID); err != nil {
		_ = h.groupRepo.RemoveMember(invitation.GroupID, studentID)
		_ = h.invitationRepo.SetStatus(invitationID, models.InvitationPending)
		http.Error(w, "Error al cancelar tus otras invitaciones", http.StatusInternalServerError)
		return
	}
	if err := h.studentRepo.SetGroupMembership(studentID, invitation.GroupID, false); err != nil {
		_ = h.groupRepo.RemoveMember(invitation.GroupID, studentID)
		_ = h.invitationRepo.SetStatus(invitationID, models.InvitationPending)
		http.Error(w, "Error al incorporarte al grupo", http.StatusInternalServerError)
		return
	}

	views, err := h.publicGroups([]models.Group{*group})
	if err != nil || len(views) == 0 {
		http.Error(w, "Error al obtener el grupo", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, views[0])
}

// Reject handles POST /api/groups/invitations/{id}/reject (token de alumno).
func (h *GroupHandler) Reject(w http.ResponseWriter, r *http.Request) {
	if !h.groupsOpen() {
		http.Error(w, "La formación de grupos está cerrada", http.StatusForbidden)
		return
	}
	studentID, err := StudentIDFromRequest(r)
	if err != nil {
		http.Error(w, "Identificación inválida", http.StatusUnauthorized)
		return
	}
	invitationID, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	invitation, err := h.invitationRepo.FindPendingByID(invitationID)
	if err != nil {
		http.Error(w, "Invitación no encontrada o ya respondida", http.StatusNotFound)
		return
	}
	if invitation.StudentID != studentID {
		http.Error(w, "Esta invitación no es para vos", http.StatusForbidden)
		return
	}
	if err := h.invitationRepo.SetStatus(invitationID, models.InvitationRejected); err != nil {
		http.Error(w, "Error al rechazar la invitación", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"message": "Invitación rechazada"})
}

// CancelInvitation permite al líder retirar una invitación pendiente.
func (h *GroupHandler) CancelInvitation(w http.ResponseWriter, r *http.Request) {
	if !h.groupsOpen() {
		http.Error(w, "La formación de grupos está cerrada", http.StatusForbidden)
		return
	}
	studentID, err := StudentIDFromRequest(r)
	if err != nil {
		http.Error(w, "Identificación inválida", http.StatusUnauthorized)
		return
	}
	invitationID, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	invitation, err := h.invitationRepo.FindPendingByID(invitationID)
	if err != nil {
		http.Error(w, "Invitación no encontrada o ya respondida", http.StatusNotFound)
		return
	}
	group, err := h.groupRepo.FindByID(invitation.GroupID)
	if err != nil || group.LeaderID != studentID {
		http.Error(w, "No tenés permiso para cancelar esta invitación", http.StatusForbidden)
		return
	}
	if err := h.invitationRepo.SetStatus(invitationID, models.InvitationCanceled); err != nil {
		http.Error(w, "Error al cancelar la invitación", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, map[string]string{"message": "Invitación cancelada"})
}

// DeleteAdminGroup handles DELETE /api/admin/groups/{id} (protegido).
func (h *GroupHandler) DeleteAdminGroup(w http.ResponseWriter, r *http.Request) {
	id, err := primitive.ObjectIDFromHex(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	group, err := h.groupRepo.FindByID(id)
	if err != nil {
		http.Error(w, "Grupo no encontrado", http.StatusNotFound)
		return
	}

	_ = h.invitationRepo.CancelByGroup(id)
	_ = h.studentRepo.ClearGroup(group.LeaderID)
	for _, memberID := range group.MemberIDs {
		_ = h.studentRepo.ClearGroup(memberID)
	}
	if err := h.groupRepo.Delete(id); err != nil {
		http.Error(w, "Error al eliminar el grupo", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// RemoveMember handles DELETE /api/admin/groups/{id}/members/{memberId}.
func (h *GroupHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	memberID, err := primitive.ObjectIDFromHex(vars["memberId"])
	if err != nil {
		http.Error(w, "ID inválido", http.StatusBadRequest)
		return
	}
	if err := h.groupRepo.RemoveMember(groupID, memberID); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			http.Error(w, "Integrante no encontrado en el grupo", http.StatusNotFound)
			return
		}
		http.Error(w, "Error al quitar el integrante", http.StatusInternalServerError)
		return
	}
	_ = h.studentRepo.ClearGroup(memberID)

	group, err := h.groupRepo.FindByID(groupID)
	if err != nil {
		http.Error(w, "Error al obtener el grupo", http.StatusInternalServerError)
		return
	}
	views, err := h.publicGroups([]models.Group{*group})
	if err != nil || len(views) == 0 {
		http.Error(w, "Error al obtener el grupo", http.StatusInternalServerError)
		return
	}
	jsonResponse(w, views[0])
}

func jsonResponse(w http.ResponseWriter, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(value)
}
