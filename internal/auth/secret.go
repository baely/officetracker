package auth

import "math/rand"

// GenerateSecret generates a random 24 alphanumeric secret
func GenerateSecret() string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890+/")
	b := make([]rune, 64)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return "officetracker:" + string(b)
}
