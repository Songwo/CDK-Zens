package main

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
)

func main() {
	secret := []byte("zf3sg4ikVorzR1S/cel4+o/VkQH+4LZxU8RkX0c0Ne6BFPf9NnPL09AoHSK9Zg95")
	ssoToken := "eyJhbGciOiJIUzUxMiJ9.eyJzY2hvb2wiOiLmnKzmoKEiLCJsZXZlbCI6Miwicm9sZXMiOlsiUk9MRV9TVVBFUl9BRE1JTiJdLCJuaWNrbmFtZSI6InplbnMiLCJhdmF0YXIiOiJodHRwczovL2FsbGluc29uZy50b3Avc3RhdGljL2F2YXRhci85NTI1OTQ2My1hNjQ3LTQ4ZjYtOWZkOS0zNWQ2YTczM2RkMDIuanBnIiwiZW1haWwiOiJoaXJsZXkzQHNvbmdib2tlLnVzLmtnIiwic3NvIjp0cnVlLCJjbGllbnRfaWQiOiJjZGstYWlyZHJvcCIsInVzZXJuYW1lIjoiU29uZyIsInN1YiI6IjIwMjcyMzExNjczMDcyMTQ4NTAiLCJpYXQiOjE3Nzg3Mjg3MTYsImV4cCI6MTc3ODcyOTAxNn0.P5AYDqbMvmKAK_SL1tJecVHolPFmyCeSrg2mpkU4RC-KXdsKmb24oR0trRbef7Yc6Y72odAsCMikfNpffaLurA"
	token, err := jwt.Parse(ssoToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Valid:", token.Valid)
	fmt.Println("Claims:", token.Claims)
}
