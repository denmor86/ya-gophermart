package validators

import (
	"strconv"
	"strings"
)

// CheckNumber проверяет строку используя алгоритм Луна
func CheckNumber(number string) bool {
	// Удаляем все пробелы
	number = strings.ReplaceAll(number, " ", "")

	// Проверяем, что строка состоит только из цифр
	if _, err := strconv.Atoi(number); err != nil {
		return false
	}

	sum := 0
	alternate := false

	// Идем по цифрам справа налево
	for i := len(number) - 1; i >= 0; i-- {
		digit, _ := strconv.Atoi(string(number[i]))

		if alternate {
			digit *= 2
			if digit > 9 {
				digit = (digit % 10) + 1
			}
		}

		sum += digit
		alternate = !alternate
	}

	// Число валидно, если сумма кратна 10
	return sum%10 == 0
}
