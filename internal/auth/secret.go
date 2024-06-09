package auth

import "math/rand"

// GenerateSecret generates a random 24 alphanumeric secret
func GenerateSecret() string {
	var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, 24)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

// up:yeah:iLjC5qpq1TVDmakxZSmBNGysW1q3sD3HunldxUCSLYHiynR1zUUMuKUAsAoVkQXkYXuAWz7Bx6ar5RQN9S6kSgEHKdtHjKDxVsJSdWCTFQRnLcqQ2FjKY75s4GqbAUCb
