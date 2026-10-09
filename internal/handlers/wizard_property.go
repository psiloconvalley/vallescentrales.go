package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"vallescentrales/internal/models"
	"vallescentrales/internal/repo"
	"vallescentrales/internal/services"
	"vallescentrales/internal/slug"
	"vallescentrales/internal/wizard"
)

type WizardPropertyHandler struct {
	engine   *wizard.Engine
	repo     *repo.ListingRepo
	renderer Renderer
	storage  *services.StorageService
}

func NewWizardPropertyHandler(
	engine *wizard.Engine,
	repo *repo.ListingRepo,
	renderer Renderer,
	storage *services.StorageService,
) *WizardPropertyHandler {
	return &WizardPropertyHandler{
		engine:   engine,
		repo:     repo,
		renderer: renderer,
		storage:  storage,
	}
}

func (h *WizardPropertyHandler) Step(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login?next=/publicar", http.StatusSeeOther)
		return
	}

	stepStr := chi.URLParam(r, "step")
	stepNum, err := strconv.Atoi(stepStr)
	if err != nil || stepNum < 1 || stepNum > 3 {
		stepNum = 1
	}

	session, err := h.engine.StartOrResume(r.Context(), user.ID, wizard.FlowPropertyPublish)
	if err != nil {
		http.Error(w, "Error cargando la sesión", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Session":        session,
		"CurrentStep":    stepNum,
		"TotalSteps":     3,
		"StepName":       session.StepName(stepNum),
		"StepTitle":      wizard.PropertyStepTitles[stepNum],
		"StepSubtitle":   wizard.PropertyStepSubtitles[stepNum],
		"StepData":       session.GetStepData(stepNum),
		"Municipalities": wizard.OaxacaMunicipalities,
		"User":           user,
	}

	h.renderer.Render(w, r, fmt.Sprintf("wizard/property/step%d", stepNum), data)
}

func (h *WizardPropertyHandler) SaveStep(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Error(w, "No autorizado", http.StatusUnauthorized)
		return
	}

	stepStr := chi.URLParam(r, "step")
	stepNum, _ := strconv.Atoi(stepStr)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		_ = r.ParseForm()
	}

	formData := formToMap(r.Form)
	session, err := h.engine.SaveStep(r.Context(), user.ID, wizard.FlowPropertyPublish, stepNum, formData)
	if err != nil {
		http.Error(w, "Error al guardar", http.StatusInternalServerError)
		return
	}

	if stepNum < session.TotalSteps {
		http.Redirect(w, r, fmt.Sprintf("/publicar/paso/%d", stepNum+1), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/publicar/completar", http.StatusSeeOther)
}

func (h *WizardPropertyHandler) AutoSave(w http.ResponseWriter, r *http.Request) {
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

	if err := h.engine.AutoSave(r.Context(), user.ID, wizard.FlowPropertyPublish, stepNum, payload); err != nil {
		http.Error(w, `{"error":"save_failed"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"saved"}`))
}

func (h *WizardPropertyHandler) UploadPhoto(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, `{"error":"file_too_large"}`, http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("photo")
	if err != nil {
		http.Error(w, `{"error":"no_file"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	res, err := h.storage.Upload(r.Context(), "properties", header.Filename, header.Header.Get("Content-Type"), file)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"url": res.PublicURL,
	})
}

func (h *WizardPropertyHandler) Complete(w http.ResponseWriter, r *http.Request) {
	user := getUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	session, err := h.engine.Get(r.Context(), user.ID, wizard.FlowPropertyPublish)
	if err != nil || session == nil {
		http.Redirect(w, r, "/publicar", http.StatusSeeOther)
		return
	}

	d := session.Data
	title := stringVal(d, "title")
	price, _ := strconv.ParseFloat(stringVal(d, "price"), 64)
	lotSize, _ := strconv.ParseFloat(stringVal(d, "lot_size_m2"), 64)
	constrSize, _ := strconv.ParseFloat(stringVal(d, "construction_m2"), 64)
	bedrooms, _ := strconv.Atoi(stringVal(d, "bedrooms"))
	bathrooms, _ := strconv.ParseFloat(stringVal(d, "bathrooms"), 64)
	parking, _ := strconv.Atoi(stringVal(d, "parking_spaces"))

	now := time.Now().UTC()

	var descPtr *string
	if desc := stringVal(d, "description"); desc != "" {
		descPtr = &desc
	}
	var addrPtr *string
	if addr := stringVal(d, "address"); addr != "" {
		addrPtr = &addr
	}
	var commPtr *string
	if comm := stringVal(d, "neighborhood"); comm != "" {
		commPtr = &comm
	}
	var bedPtr *int
	if bedrooms > 0 {
		bedPtr = &bedrooms
	}
	var bathPtr *float64
	if bathrooms > 0 {
		bathPtr = &bathrooms
	}
	var parkPtr *int
	if parking > 0 {
		parkPtr = &parking
	}
	var lotPtr *float64
	if lotSize > 0 {
		lotPtr = &lotSize
	}
	var constrPtr *float64
	if constrSize > 0 {
		constrPtr = &constrSize
	}

	prop := &models.Property{
		ID:            uuid.New(),
		OwnerID:       user.ID,
		Title:         title,
		Slug:          slug.Generate(title),
		Description:   descPtr,
		OperationType: models.OperationType(stringVal(d, "operation_type")),
		PropertyType:  models.PropertyType(stringVal(d, "property_type")),
		LegalRegime:   models.LegalRegime(stringVal(d, "legal_regime")),
		PriceMXN:      price,
		Currency:      "MXN",
		AreaTotalM2:   lotPtr,
		AreaBuiltM2:   constrPtr,
		Bedrooms:      bedPtr,
		Bathrooms:     bathPtr,
		ParkingSpaces: parkPtr,
		Municipality:  stringVal(d, "municipality"),
		Community:     commPtr,
		Address:       addrPtr,
		Status:        models.StatusActive,
		ListingTier:   "free",
		ShowPhone:     user.ShowPhone,
		ShowWhatsApp:  user.ShowWhatsApp,
		CreatedAt:     now,
		UpdatedAt:     now,
		PublishedAt:   &now,
	}

	photos := stringSliceVal(d, "photos")
	if err := h.repo.CreateProperty(r.Context(), prop, photos); err != nil {
		http.Error(w, "Error al publicar propiedad", http.StatusInternalServerError)
		return
	}

	_ = h.engine.Finalize(r.Context(), user.ID, wizard.FlowPropertyPublish)

	http.Redirect(w, r, fmt.Sprintf("/propiedad/%s?publicado=1", prop.Slug), http.StatusSeeOther)
}
