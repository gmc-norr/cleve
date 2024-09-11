package mongo

import "fmt"

type PageOutOfBoundsError struct {
	page       int
	totalPages int
}

func (e PageOutOfBoundsError) Error() string {
	return fmt.Sprintf("invalid page number %d for a result of total %d pages", e.page, e.totalPages)
}
