// internal/catalog/municipalities.go
// Canonical list of Oaxacan municipalities and regions.
// Defined once. Used in templates, validation, and search.
// ADR-005: domain vocabulary must be consistent everywhere.

package catalog

// Municipality represents a selectable location.
type Municipality struct {
	Value  string // stored in DB
	Label  string // displayed to users
	Region string // for grouping in UI
}

// Municipalities is the canonical list of all supported locations.
// Organized by region for dropdown grouping.
var Municipalities = []Municipality{
	// Valles Centrales
	{Value: "oaxaca_de_juarez", Label: "Oaxaca de Juárez", Region: "Valles Centrales"},
	{Value: "tlacolula_de_matamoros", Label: "Tlacolula de Matamoros", Region: "Valles Centrales"},
	{Value: "mitla", Label: "Mitla", Region: "Valles Centrales"},
	{Value: "san_pablo_villa_de_mitla", Label: "San Pablo Villa de Mitla", Region: "Valles Centrales"},
	{Value: "etla", Label: "Villa de Etla", Region: "Valles Centrales"},
	{Value: "san_agustin_etla", Label: "San Agustín Etla", Region: "Valles Centrales"},
	{Value: "zimatlan", Label: "Zimatlán de Álvarez", Region: "Valles Centrales"},
	{Value: "ocotlan_de_morelos", Label: "Ocotlán de Morelos", Region: "Valles Centrales"},
	{Value: "zaachila", Label: "Villa de Zaachila", Region: "Valles Centrales"},
	{Value: "cuilapam_de_guerrero", Label: "Cuilápam de Guerrero", Region: "Valles Centrales"},
	{Value: "san_bartolo_coyotepec", Label: "San Bartolo Coyotepec", Region: "Valles Centrales"},
	{Value: "san_dionisio_ocotepec", Label: "San Dionisio Ocotepec", Region: "Valles Centrales"},
	{Value: "santa_cruz_xoxocotlan", Label: "Santa Cruz Xoxocotlán", Region: "Valles Centrales"},
	{Value: "san_antonio_de_la_cal", Label: "San Antonio de la Cal", Region: "Valles Centrales"},
	{Value: "santa_lucia_del_camino", Label: "Santa Lucía del Camino", Region: "Valles Centrales"},
	{Value: "santa_maria_atzompa", Label: "Santa María Atzompa", Region: "Valles Centrales"},
	{Value: "tlalixtac_de_cabrera", Label: "Tlalixtac de Cabrera", Region: "Valles Centrales"},
	{Value: "san_jeronimo_tlacochahuaya", Label: "San Jerónimo Tlacochahuaya", Region: "Valles Centrales"},
	{Value: "teotitlan_del_valle", Label: "Teotitlán del Valle", Region: "Valles Centrales"},

	// Costa — Huatulco & Surroundings
	{Value: "santa_maria_huatulco", Label: "Santa María Huatulco", Region: "Costa"},
	{Value: "san_pedro_pochutla", Label: "San Pedro Pochutla", Region: "Costa"},
	{Value: "santa_maria_tonameca", Label: "Santa María Tonameca (Mazunte/Zipolite)", Region: "Costa"},
	{Value: "san_pedro_mixtepec", Label: "San Pedro Mixtepec (Puerto Escondido)", Region: "Costa"},
	{Value: "santa_maria_colotepec", Label: "Santa María Colotepec (Playa Carrizalillo)", Region: "Costa"},
	{Value: "san_miguel_del_puerto", Label: "San Miguel del Puerto", Region: "Costa"},
	{Value: "santiago_astata", Label: "Santiago Astata", Region: "Costa"},

	// Istmo
	{Value: "juchitan_de_zaragoza", Label: "Juchitán de Zaragoza", Region: "Istmo"},
	{Value: "salina_cruz", Label: "Salina Cruz", Region: "Istmo"},
	{Value: "santo_domingo_tehuantepec", Label: "Santo Domingo Tehuantepec", Region: "Istmo"},
	{Value: "san_blas_atempa", Label: "San Blas Atempa", Region: "Istmo"},

	// Sierra Norte
	{Value: "ixtlan_de_juarez", Label: "Ixtlán de Juárez", Region: "Sierra Norte"},
	{Value: "capulalpam_de_mendez", Label: "Capulálpam de Méndez", Region: "Sierra Norte"},
	{Value: "guelatao_de_juarez", Label: "Guelatao de Juárez", Region: "Sierra Norte"},
	{Value: "santa_catarina_lachatao", Label: "Santa Catarina Lachatao", Region: "Sierra Norte"},

	// Sierra Sur
	{Value: "miahuatlan_de_porfirio_diaz", Label: "Miahuatlán de Porfirio Díaz", Region: "Sierra Sur"},
	{Value: "san_jose_del_pacifico", Label: "San José del Pacífico", Region: "Sierra Sur"},

	// Mixteca
	{Value: "huajuapan_de_leon", Label: "Huajuapan de León", Region: "Mixteca"},
	{Value: "tlaxiaco", Label: "Heroica Ciudad de Tlaxiaco", Region: "Mixteca"},
	{Value: "nochixtlan", Label: "Asunción Nochixtlán", Region: "Mixteca"},

	// Cañada
	{Value: "teotitlan_de_flores_magon", Label: "Teotitlán de Flores Magón", Region: "Cañada"},
	{Value: "cuicatlan", Label: "San Juan Bautista Cuicatlán", Region: "Cañada"},

	// Papaloapan
	{Value: "tuxtepec", Label: "San Juan Bautista Tuxtepec", Region: "Papaloapan"},
	{Value: "loma_bonita", Label: "Loma Bonita", Region: "Papaloapan"},
}

// ValidMunicipality returns true if the value is in the canonical list.
func ValidMunicipality(value string) bool {
	for _, m := range Municipalities {
		if m.Value == value {
			return true
		}
	}
	return false
}

// MunicipalityLabel returns the display label for a municipality value.
func MunicipalityLabel(value string) string {
	for _, m := range Municipalities {
		if m.Value == value {
			return m.Label
		}
	}
	return value
}

// Regions returns unique region names in display order.
func Regions() []string {
	return []string{
		"Valles Centrales",
		"Costa",
		"Istmo",
		"Sierra Norte",
		"Sierra Sur",
		"Mixteca",
		"Cañada",
		"Papaloapan",
	}
}

// MunicipalitiesByRegion returns municipalities grouped by region.
func MunicipalitiesByRegion() map[string][]Municipality {
	grouped := make(map[string][]Municipality)
	for _, m := range Municipalities {
		grouped[m.Region] = append(grouped[m.Region], m)
	}
	return grouped
}
