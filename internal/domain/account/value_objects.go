package account

import (
	"fmt"
	"strconv"
)

type AccountStatus string

const (
	Active   AccountStatus = "ACTIVE"
	Inactive AccountStatus = "INACTIVE"
)

// Money represents an amount in cents to avoid floating-point precision issues.
type Money int64

func NewMoney(cents int64) (Money, error) {
	if cents < 0 {
		return 0, ErrInvalidAmount
	}
	return Money(cents), nil
}

func (m Money) Cents() int64 {
	return int64(m)
}

func (m Money) IsPositive() bool {
	return m > 0
}

func (m Money) Add(other Money) Money {
	return m + other
}

func (m Money) Sub(other Money) (Money, error) {
	if m < other {
		return 0, ErrInsufficientFunds
	}
	return m - other, nil
}

func (m Money) String() string {
	reais := m.Cents() / 100
	centavos := m.Cents() % 100
	return fmt.Sprintf("R$ %d.%02d", reais, centavos)
}

// CPF is a value object that validates the Brazilian CPF document.
type CPF string

func NewCPF(value string) (CPF, error) {
	digits := onlyDigits(value)
	if len(digits) != 11 {
		return "", ErrInvalidCPF
	}
	if allSame(digits) {
		return "", ErrInvalidCPF
	}
	if !validCPFCheckDigits(digits) {
		return "", ErrInvalidCPF
	}
	return CPF(digits), nil
}

func (c CPF) String() string {
	return string(c)
}

func onlyDigits(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			out = append(out, s[i])
		}
	}
	return string(out)
}

func allSame(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}

func validCPFCheckDigits(digits string) bool {
	d := make([]int, 11)
	for i, ch := range digits {
		n, _ := strconv.Atoi(string(ch))
		d[i] = n
	}

	sum := 0
	for i := 0; i < 9; i++ {
		sum += d[i] * (10 - i)
	}
	r1 := (sum * 10) % 11
	if r1 == 10 || r1 == 11 {
		r1 = 0
	}
	if r1 != d[9] {
		return false
	}

	sum = 0
	for i := 0; i < 10; i++ {
		sum += d[i] * (11 - i)
	}
	r2 := (sum * 10) % 11
	if r2 == 10 || r2 == 11 {
		r2 = 0
	}
	return r2 == d[10]
}
