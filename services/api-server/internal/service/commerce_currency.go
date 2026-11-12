package service

import (
	"context"
	"database/sql"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
)

type currencyReader interface {
	QueryRowContext(context.Context, repository.Statement, ...any) *sql.Row
}

func currencySnapshot(ctx context.Context, reader currencyReader, uid string) (Object, error) {
	values := make([]sql.NullInt64, len(currencyFields))
	dest := make([]any, len(values))
	for i := range values {
		dest[i] = &values[i]
	}
	if err := reader.QueryRowContext(ctx, repository.ReadCurrency, uid).Scan(dest...); err != nil {
		return nil, err
	}
	out := Object{}
	for i, key := range currencyFields {
		out[key] = values[i].Int64
	}
	return out, nil
}
