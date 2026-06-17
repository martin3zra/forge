package foundation

import (
	"fmt"
	"strings"
)

var units = []string{
	"", "uno", "dos", "tres", "cuatro",
	"cinco", "seis", "siete", "ocho", "nueve",
	"diez", "once", "doce", "trece", "catorce",
	"quince", "dieciseis", "diecisiete", "dieciocho", "diecinueve",
}

var tens = []string{
	"", "", "veinte", "treinta", "cuarenta",
	"cincuenta", "sesenta", "setenta", "ochenta", "noventa",
}

var hundreds = []string{
	"", "ciento", "doscientos", "trescientos", "cuatrocientos",
	"quinientos", "seiscientos", "setecientos", "ochocientos", "novecientos",
}

func numberToWords(n int64) string {
	switch {
	case n == 0:
		return "cero"

	case n < 20:
		return units[n]

	case n < 100:
		if n == 20 {
			return "veinte"
		}
		if n < 30 {
			return "veinti" + units[n-20]
		}
		if n%10 == 0 {
			return tens[n/10]
		}
		return tens[n/10] + " y " + units[n%10]

	case n == 100:
		return "cien"

	case n < 1000:
		rest := n % 100
		if rest == 0 {
			return hundreds[n/100]
		}
		return hundreds[n/100] + " " + numberToWords(rest)

	case n < 1_000_000:
		thousands := n / 1000
		rest := n % 1000

		var text string
		if thousands == 1 {
			text = "mil"
		} else {
			text = numberToWords(thousands) + " mil"
		}

		if rest == 0 {
			return text
		}
		return text + " " + numberToWords(rest)

	case n < 1_000_000_000:
		millions := n / 1_000_000
		rest := n % 1_000_000

		var text string
		if millions == 1 {
			text = "un millon"
		} else {
			text = numberToWords(millions) + " millones"
		}

		if rest == 0 {
			return text
		}
		return text + " " + numberToWords(rest)

	default:
		return "cantidad fuera de rango"
	}
}

func MoneyToText(cents int64, currency string) string {
	entero := cents / 100
	decimals := cents % 100

	text := strings.ToUpper(numberToWords(entero))
	return fmt.Sprintf(
		"%s %s CON %02d/100",
		text,
		strings.ToUpper(currency),
		decimals,
	)
}
