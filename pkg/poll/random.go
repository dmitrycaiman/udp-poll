package poll

import "math/rand"

var symbols = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

func randomPayload(i int) []byte {
	result := make([]byte, i+rand.Intn(i))
	for i := range result {
		result[i] = symbols[rand.Intn(len(symbols))]
	}
	return result
}
