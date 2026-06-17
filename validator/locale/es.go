package locale

func SpanishMessages() map[string]any {
	return map[string]any{
		"required": "El campo %s es obligatorio.",
		"max": map[string]any{
			"int":    "El campo %s no debe ser mayor que %v.",
			"string": "El campo %s no debe tener más de %v caracteres.",
			"slice":  "El campo %s no debe tener más de %v elementos.",
		},
		"min": map[string]any{
			"int":    "El campo %s debe ser al menos %v.",
			"string": "El campo %s debe tener al menos %v caracteres.",
			"slice":  "El campo %s debe tener al menos %v elementos.",
		},
		"gte": map[string]any{
			"int":    "El campo %s debe ser mayor o igual a %v.",
			"string": "El campo %s debe ser mayor o igual a %v caracteres.",
		},
		"gt": map[string]any{
			"int":    "El campo %s debe ser mayor que %v.",
			"string": "El campo %s debe tener más de %v caracteres.",
		},
		"lte": map[string]any{
			"int":    "El campo %s debe ser menor o igual a %v.",
			"string": "El campo %s debe ser menor o igual a %v caracteres.",
		},
		"lt": map[string]any{
			"int":    "El campo %s debe ser menor o igual a %v.",
			"string": "El campo %s debe tener menos de %v caracteres.",
		},
		"between": map[string]any{
			"int":    "El campo %s debe estar entre %v y %v.",
			"string": "El campo %s debe tener entre %v y %v caracteres.",
		},
		"different":        "El campo %s y %v deben ser diferentes.",
		"email":            "El campo %s debe ser una dirección de correo electrónico válida.",
		"exists":           "El %s seleccionado no es válido.",
		"unique":           "El %s ya ha sido tomado.",
		"current_password": "La contraseña es incorrecta.",
		"in":               "El %s seleccionado no es válido.",
		"lowercase":        "El campo %s debe estar en minúsculas.",
		"uppercase":        "El campo %s debe estar en mayúsculas.",
		"date":             "El campo %s debe ser una fecha válida.",
		"after":            "El campo %s debe ser una fecha posterior a %v.",
		"digits":           "El campo %s debe tener %v dígitos.",
		"digits_between":   "El campo %s debe estar entre %v y %v dígitos.",
		"max_digits":       "El campo %s no debe tener más de %v dígitos.",
		"min_digits":       "El campo %s debe tener al menos %v dígitos.",
	}
}
