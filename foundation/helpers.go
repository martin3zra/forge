package foundation

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"reflect"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func GetIpAddress(r *http.Request) (string, error) {
	ips := r.Header.Get("X-Forwarded-For")
	splitIps := strings.Split(ips, ",")
	if len(splitIps) > 0 {
		// get last IP in list since ELB prepends other user defined IPs, meaning the last one is the actual client IP.
		netIP := net.ParseIP(splitIps[len(splitIps)-1])
		if netIP != nil {
			return netIP.String(), nil
		}
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	netIP := net.ParseIP(ip)
	if netIP != nil {
		ip := netIP.String()
		if ip == "::1" {
			return "127.0.0.1", nil
		}
		return ip, nil
	}

	return "", errors.New("IP not found")
}

func AsMap(obj any) map[string]any {
	result := make(map[string]any)
	val := reflect.ValueOf(obj)
	typ := reflect.TypeOf(obj)

	if val.Kind() == reflect.Ptr {
		val = val.Elem()
		typ = typ.Elem()
	}

	for i := range val.NumField() {
		field := typ.Field(i)
		if field.IsExported() {
			result[field.Name] = val.Field(i).Interface()
		}
	}

	return result
}

func GeneratePrefixedNumber(prefix string, length, value int) string {
	return fmt.Sprintf(fmt.Sprintf("%s%%0%dv", prefix, length), value)
}

func ToJSON(m any) string {
	js, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}

	return strings.ReplaceAll(string(js), ",", ", ")
}

func AsJSON(m any) []byte {
	js, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}

	return js
}

func MapSlice[T, U any](s []T, f func(T) U) []U {
	result := make([]U, len(s))
	for i, v := range s {
		result[i] = f(v)
	}
	return result
}

func ResolveError(err error) error {
	for {
		unw := errors.Unwrap(err)
		if unw == nil {
			return err
		}
		err = unw
	}
}

type AmountFormat struct {
	Amount float64
	Symbol string
}

func FormatAmount(amount float64, options ...string) string {
	p := message.NewPrinter(language.English)
	symbol := "$"
	if len(options) > 0 {
		symbol = options[0]
	}
	return p.Sprintf("%s%.2f", symbol, amount) // "$1,234,567.89"
}

func AsUUID(s string) (uuid.UUID, error) {
	if s == "" {
		return uuid.Nil, nil
	}
	u, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil, err
	}
	return u, nil
}
