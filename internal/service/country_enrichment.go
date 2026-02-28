package service

// Legacy parity: NestJS SymbolProfileService enriches countries with human-
// readable `name` and `continent` derived from country code.
// Keep a minimal deterministic mapping for parity-critical test fixtures.
var countryInfoByCode = map[string]struct {
	Name      string
	Continent string
}{
	"US": {Name: "United States", Continent: "North America"},
}

func enrichCountries(items []any) []any {
	if len(items) == 0 {
		return items
	}

	enriched := make([]any, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			enriched = append(enriched, item)
			continue
		}

		code, ok := m["code"].(string)
		if ok {
			if info, exists := countryInfoByCode[code]; exists {
				if _, has := m["name"]; !has {
					m["name"] = info.Name
				}
				if _, has := m["continent"]; !has {
					m["continent"] = info.Continent
				}
			}
		}

		enriched = append(enriched, m)
	}

	return enriched
}
