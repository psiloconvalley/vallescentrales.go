package wizard

import (
	"time"

	"github.com/google/uuid"
)

type FlowType string

const (
	FlowAccountSetup    FlowType = "account_setup"
	FlowPropertyPublish FlowType = "property_publish"

	// Aliases for compatibility
	FlowAccount  FlowType = "account_setup"
	FlowProperty FlowType = "property_publish"
)

// Session represents the persisted multi-step state in PostgreSQL.
type Session struct {
	ID          uuid.UUID              `json:"id" db:"id"`
	UserID      uuid.UUID              `json:"user_id" db:"user_id"`
	FlowType    FlowType               `json:"flow_type" db:"flow_type"`
	CurrentStep int                    `json:"current_step" db:"current_step"`
	TotalSteps  int                    `json:"total_steps" db:"total_steps"`
	Data        map[string]interface{} `json:"data" db:"data"`
	CreatedAt   time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at" db:"updated_at"`
	ExpiresAt   time.Time              `json:"expires_at" db:"expires_at"`
}

func (s *Session) StepName(step int) string {
	if s.FlowType == FlowAccountSetup {
		switch step {
		case 1:
			return "Identidad y Perfil"
		case 2:
			return "Contacto y Privacidad"
		}
	}
	if s.FlowType == FlowPropertyPublish {
		switch step {
		case 1:
			return "Tipo y Régimen Legal"
		case 2:
			return "Detalles y Precio"
		case 3:
			return "Fotografías"
		}
	}
	return "Paso"
}

func (s *Session) GetStepData(step int) map[string]interface{} {
	if s.Data == nil {
		return make(map[string]interface{})
	}
	return s.Data
}

var AccountStepTitles = map[int]string{
	1: "Configura tu Perfil",
	2: "Información de Contacto",
}

var AccountStepSubtitles = map[int]string{
	1: "Define cómo te verán los compradores y vendedores en los Valles Centrales.",
	2: "Elige qué datos de contacto mostrar públicamente en tus anuncios.",
}

var PropertyStepTitles = map[int]string{
	1: "¿Qué tipo de propiedad deseas publicar?",
	2: "Detalles, Ubicación y Precio",
	3: "Fotografías de la Propiedad",
}

var PropertyStepSubtitles = map[int]string{
	1: "Selecciona el régimen de tenencia de la tierra y la operación.",
	2: "Proporciona medidas precisas y ubicación exacta en los Valles Centrales.",
	3: "Sube imágenes claras para atraer más compradores interesados.",
}

var OaxacaMunicipalities = []string{
	"Oaxaca de Juárez",
	"San Felipe del Agua",
	"Santa Cruz Xoxocotlán",
	"San Antonio de la Cal",
	"Santa Lucía del Camino",
	"San Sebastián Tutla",
	"Santa María del Tule",
	"Tlalixtac de Cabrera",
	"San Agustín Yatareni",
	"San Andrés Huayápam",
	"Santa María Atzompa",
	"San Jacinto Amilpas",
	"San Pablo Etla",
	"Villa de Etla",
	"San Agustín Etla",
	"Guadalupe Etla",
	"Nazareno Etla",
	"Reyes Etla",
	"Santo Domingo Tomaltepec",
	"Villa de Zaachila",
	"Cuilápam de Guerrero",
	"San Bartolo Coyotepec",
	"Santa María Coyotepec",
	"San Raymundo Jalpan",
	"Ocotlán de Morelos",
	"Tlacolula de Matamoros",
	"San Jerónimo Tlacochahuaya",
	"Villa de Mitla",
	"Zimatlán de Álvarez",
}
