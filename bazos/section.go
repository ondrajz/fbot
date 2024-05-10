package bazos

type AdSection struct {
	Category string
	Section  string
}

const (
	AnySection = "www"
)

type Section string

func (s Section) String() string {
	if n, ok := sectionsNames[string(s)]; ok {
		return n
	}
	return string(s)
}

func AllSections() []string {
	return sectionsList
}

// Sections (Rubriky)
var sectionsList = []string{
	AnySection,
	"auto",
	"deti",
	"dom",
	"elektro",
	"foto",
	"hudba",
	"knihy",
	"mobil",
	"motocykle",
	"nabytok",
	"oblecenie",
	"pc",
	"praca",
	"reality",
	"sluzby",
	"stroje",
	"sport",
	"vstupenky",
	"zvierata",
	"ostatne",
}

var sectionsNames = map[string]string{
	AnySection:  "Všetky rubriky",
	"auto":      "Auto",
	"deti":      "Deti",
	"dom":       "Dom a záhrada",
	"elektro":   "Elektro",
	"foto":      "Foto",
	"hudba":     "Hudba",
	"knihy":     "Knihy",
	"mobil":     "Mobily",
	"motocykle": "Motocykle",
	"nabytok":   "Nábytok",
	"oblecenie": "Oblečenie",
	"pc":        "PC",
	"praca":     "Práca",
	"reality":   "Reality",
	"sluzby":    "Služby",
	"stroje":    "Stroje",
	"sport":     "Šport",
	"vstupenky": "Vstupenky",
	"zvierata":  "Zvieratá",
	"ostatne":   "Ostatné",
}
