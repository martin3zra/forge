package locale

func EnglishMessages() map[string]any {
	return map[string]any{
		"required": "The %s field is required.",
		"max": map[string]any{
			"int":    "The %s field must not be greater than %v.",
			"string": "The %s field must not be greater than %v characters.",
			"slice":  "The %s field must not have more than %v items.",
		},
		"min": map[string]any{
			"int":    "The %s field must be at least %v.",
			"string": "The %s field must be at least %v characters.",
			"slice":  "The %s field must has at least %v items.",
		},
		"gte": map[string]any{
			"int":    "The %s field must be greater than or equal to %v.",
			"string": "The %s field must be greater than or equal to %v characters.",
		},
		"gt": map[string]any{
			"int":    "The %s field must be greater than %v.",
			"string": "The %s field must be greater than %v characters.",
		},
		"lte": map[string]any{
			"int":    "The %s field must be less than or equal to %v.",
			"string": "The %s field must be less than or equal to %v characters.",
		},
		"lt": map[string]any{
			"int":    "The %s field must be less than or equal to %v.",
			"string": "The %s field must be less than %v characters.",
		},
		"between": map[string]any{
			"int":    "The %s field must be between %v and %v.",
			"string": "The %s field must be between %v and %v characters.",
		},
		"different":        "The %s field and %v must be different.",
		"email":            "The %s field must be a valid email address.",
		"exists":           "The selected %s is invalid.",
		"unique":           "The %s has already been taken.",
		"current_password": "The password is incorrect.",
		"in":               "The selected %s is invalid.",
		"lowercase":        "The %s field must be lowercase.",
		"uppercase":        "The %s field must be uppercase.",
		"date":             "The %s field must be a valid date.",
		"after":            "The %s field must be a date after %v.",
		"digits":           "The %s field must be %v digits.",
		"digits_between":   "The %s field must be between %v and %v digits.",
		"max_digits":       "The %s field must not have more than %v digits.",
		"min_digits":       "The %s field must have at least %v digits.",
	}
}
