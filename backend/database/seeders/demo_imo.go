package seeders

import (
	"fmt"
	"math/rand/v2"

	"github.com/suwahadi/seafarerscrew/backend/internal/service"
)

const (
	imoBaseMin = 800000
	imoBaseMax = 999999
)

func imoWithCheckDigit(base int) (string, error) {
	if base < 100000 || base > 999999 {
		return "", fmt.Errorf("imo base %d is not six digits", base)
	}
	sum := 0
	for i, weight := 0, 7; i < 6; i, weight = i+1, weight-1 {
		digit := (base / pow10(5-i)) % 10
		sum += digit * weight
	}
	imo := fmt.Sprintf("%06d%d", base, sum%10)
	if !service.ValidIMO(imo) {
		return "", fmt.Errorf("generated imo %s failed validation", imo)
	}
	return imo, nil
}

func pow10(n int) int {
	v := 1
	for range n {
		v *= 10
	}
	return v
}

func generateIMOs(rng *rand.Rand, count int) ([]string, error) {
	seen := make(map[string]bool, count)
	out := make([]string, 0, count)
	for len(out) < count {
		imo, err := imoWithCheckDigit(imoBaseMin + rng.IntN(imoBaseMax-imoBaseMin+1))
		if err != nil {
			return nil, err
		}
		if seen[imo] {
			continue
		}
		seen[imo] = true
		out = append(out, imo)
	}
	return out, nil
}
