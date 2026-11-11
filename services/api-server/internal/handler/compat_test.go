package handler

import (
	"database/sql"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/config"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/repository"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/internal/service"
	"github.com/JasonCao2003/LaServer-Go/services/api-server/migrations"
)

var Open = repository.Open
var Migrate = repository.Migrate
var checkSchema = repository.CheckSchema
var LoadConfig = config.Load
var schema = migrations.Initial

func New(db *sql.DB, cfg Config) (*API, error) {
	if err := repository.CheckSchema(db); err != nil {
		return nil, err
	}
	if err := config.ValidateRuntime(cfg); err != nil {
		return nil, err
	}
	return NewAPI(service.New(repository.New(db), &cfg), &cfg), nil
}
