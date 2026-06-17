package support

import (
	"encoding/json"
	"errors"
	"log"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/session"
)

func ParseRequest(r *http.Request, body any, params ...map[string]string) error {
	// Determine if we are dealing with a pointer
	val := reflect.ValueOf(body)
	if val.Kind() != reflect.Ptr {
		return errors.New("ParseRequest requires a pointer to a struct")
	}
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		// 1. Parse Multipart
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			return err
		}
		// 2. Hydrate struct via reflection using JSON tags
		HydrateFromForm(r, body)
	} else {

		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			log.Println(err)
			return new(foundation.BadRequest)
		}
	}

	formRequest, ok := body.(FormRequestContract)
	if ok {
		formRequest.SetContext(r.Context())
		if len(params) > 0 {
			formRequest.SetPathParams(params[0])
		}

		if !formRequest.Authorize() {
			session.GetSession(r).Errors("status", "Unauthorized")
			return foundation.Unauthorized{}
		}

		formRequest.Validate(body, formRequest.Rules(), formRequest.PrepareForValidation)
		errorMesssages := formRequest.Errors()
		if len(errorMesssages) > 0 {
			session.GetSession(r).FormErrors(foundation.ErrorBag(errorMesssages))
			return errors.New(foundation.ToJSON(errorMesssages))
		}

		formRequest.PassedValidation()

		return nil
	}

	return new(foundation.UnprocessableEntity)
}

func HydrateFromForm(r *http.Request, obj any) {
	val := reflect.ValueOf(obj)

	// We need a pointer to a struct to modify it
	if val.Kind() != reflect.Ptr || val.Elem().Kind() != reflect.Struct {
		return
	}

	val = val.Elem()
	typ := val.Type()
	fileInterface := reflect.TypeOf((*multipart.File)(nil)).Elem()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		structField := typ.Field(i)

		// Look for the "json" tag
		tag := structField.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}

		// Clean up tag (remove ",omitempty" etc.)
		tagName := strings.Split(tag, ",")[0]

		if !field.CanSet() {
			continue
		}

		// 1. Handle Files (multipart.File is an interface)
		if field.Kind() == reflect.Interface && field.Type().Implements(fileInterface) {
			file, _, err := r.FormFile(tagName)
			if err == nil {
				field.Set(reflect.ValueOf(file))
			}
			continue
		}

		// 2. Handle standard Form Values
		formValue := r.FormValue(tagName)
		if formValue == "" {
			continue
		}

		// Set the value if the field is a string
		// You can add a switch here to handle int, bool, etc.
		switch field.Kind() {
		case reflect.String:
			field.SetString(formValue)

		case reflect.Int, reflect.Int64:
			// Convert string from form to integer
			if val, err := strconv.ParseInt(formValue, 10, 64); err == nil {
				field.SetInt(val)
			}

		case reflect.Bool:
			if val, err := strconv.ParseBool(formValue); err == nil {
				field.SetBool(val)
			}

		}

	}
}
