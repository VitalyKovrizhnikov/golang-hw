package hw10programoptimization

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
)

type User struct {
	ID       int
	Name     string
	Username string
	Email    string
	Phone    string
	Password string
	Address  string
}

type DomainStat map[string]int

type emailRecord struct {
	Email string
}

func GetDomainStat(r io.Reader, domain string) (DomainStat, error) {
	decoder := json.NewDecoder(r)
	result := make(DomainStat, 1024)
	suffix := "." + domain

	for {
		var record emailRecord
		if err := decoder.Decode(&record); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return nil, fmt.Errorf("get users error: %w", err)
		}

		addDomain(result, record.Email, suffix)
	}

	return result, nil
}

func addDomain(result DomainStat, email, suffix string) {
	if !strings.Contains(email, suffix) {
		return
	}

	at := strings.IndexByte(email, '@')
	if at < 0 || at == len(email)-1 {
		return
	}

	result[strings.ToLower(email[at+1:])]++
}
