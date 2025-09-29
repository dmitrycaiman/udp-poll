package poll

import (
	"math/rand"
)

var symbols = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

// randomPayload формирует случайный набор данных длиной [n:2*n] из символов английского алфавита и цифр.
func randomPayload(n int) []byte {
	result := make([]byte, n+rand.Intn(n))
	for i := range result {
		result[i] = symbols[rand.Intn(len(symbols))]
	}
	return result
}
