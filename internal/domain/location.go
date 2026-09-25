package domain

import (
	"fmt"
	"strings"
)

type Location struct {
	ID     int64
	Name   string
	Street string
	City   string
	Active bool
}

func (l Location) Validate() error {
	if l.ID < 0 {
		return fmt.Errorf("location id must not be negative")
	}
	if strings.TrimSpace(l.Name) == "" || strings.TrimSpace(l.Street) == "" || strings.TrimSpace(l.City) == "" {
		return fmt.Errorf("location name, street, and city must not be empty")
	}
	return nil
}
