package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"vallescentrales/internal/repo"
	"vallescentrales/internal/wizard"
)

type WizardAccountHandler struct {
	engine   *wizard.Engine
	users    *repo.UserRepo
	renderer Renderer
}

func NewWizardAccountHandler(engine *wizard.Engine, users *repo.UserRepo, renderer Renderer) *WizardAccountHandler {
	return &WizardAccountHandler{
		engine:   engine,
		users:    users,
		renderer: renderer,
	}
}

func (h *WizardAccountHandler) Step(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login?next=/bienvenido", http.StatusSeeOther)
		return
	}

	stepStr := chi.URLParam(r, "step")
	stepNum, err := strconv.Atoi(stepStr)
	if err != nil || stepNum < 1 || stepNum > 2 {
		stepNum = 1
	}

	session, err := h.engine.StartOrResume(r.Context(), user.ID, wizard.FlowAccountSetup)
	if err != nil {
		http.Error(w, "Error cargando la sesión", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Session":        session,
		"CurrentStep":    stepNum,
		"TotalSteps":     2,
		"StepName":       session.StepName(stepNum),
		"StepTitle":      wizard.AccountStepTitles[stepNum],
		"StepSubtitle":   wizard.AccountStepSubtitles[stepNum],
		"StepData":       session.GetStepData(stepNum),
		"Municipalities": wizard.OaxacaMunicipalities,
		"User":           user,
	}

	h.renderer.Render(w, r, fmt.Sprintf("wizard/account/step%d", stepNum), data)
}

func (h *WizardAccountHandler) SaveStep(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	stepStr := chi.URLParam(r, "step")
	stepNum, _ := strconv.Atoi(stepStr)

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Datos inválidos", http.StatusBadRequest)
		return
	}

	formData := formToMap(r.Form)
	session, err := h.engine.SaveStep(r.Context(), user.ID, wizard.FlowAccountSetup, stepNum, formData)
	if err != nil {
		http.Error(w, "Error al guardar paso", http.StatusInternalServerError)
		return
	}

	if stepNum < session.TotalSteps {
		http.Redirect(w, r, fmt.Sprintf("/bienvenido/paso/%d", stepNum+1), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/bienvenido/completar", http.StatusSeeOther)
}

func (h *WizardAccountHandler) AutoSave(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	stepStr := chi.URLParam(r, "step")
	stepNum, _ := strconv.Atoi(stepStr)

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
		return
	}

	if err := h.engine.AutoSave(r.Context(), user.ID, wizard.FlowAccountSetup, stepNum, payload); err != nil {
		http.Error(w, `{"error":"save_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"saved"}`))
}
func (h *WizardAccountHandler) Complete(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session, err := h.engine.Get(r.Context(), user.ID, wizard.FlowAccountSetup)
	if err != nil || session == nil {
		http.Redirect(w, r, "/bienvenido", http.StatusSeeOther)
		return
	}

	d := session.Data
	fullName := stringVal(d, "full_name")
	if fullName == "" {
		fullName = user.FullName
	}

	_, err = h.users.UpdateOnboarding(
		r.Context(),
		user.ID,
		fullName,
		stringVal(d, "user_type"),
		stringPtr(stringVal(d, "phone")),
		stringPtr(stringVal(d, "whatsapp")),
		stringPtr(stringVal(d, "municipality")),
		stringPtr(stringVal(d, "bio")),
		boolVal(d, "show_phone"),
		boolVal(d, "show_whatsapp"),
	)
	if err != nil {
		http.Error(w, "Error al actualizar perfil de bienvenida", http.StatusInternalServerError)
		return
	}

	_ = h.engine.Finalize(r.Context(), user.ID, wizard.FlowAccountSetup)

	// Redirect to the correct "/cuenta" dashboard route
	http.Redirect(w, r, "/cuenta?bienvenido=1", http.StatusSeeOther)
}

