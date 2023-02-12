package seeders

import (
	"errors"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/internal/datasources/records"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
)

type Seeder interface {
	UserSeeder(userData []records.Users) (err error)
}

type seeder struct {
	db *sqlx.DB
}

func NewSeeder(db *sqlx.DB) Seeder {
	return &seeder{db: db}
}

func (s *seeder) UserSeeder(userData []records.Users) (err error) {
	if len(userData) == 0 {
		return errors.New("users data is empty")
	}

	logger.Info("inserting users data...", logrus.Fields{constants.LoggerCategory: constants.LoggerCategorySeeder})
	for _, user := range userData {
		user.CreatedAt = time.Now().In(constants.GMT7)
		if _, err = s.db.NamedQuery(`INSERT INTO users(id, username, email, password, active, role_id, created_at) VALUES (uuid_generate_v4(), :username, :email, :password, :active, :role_id, :created_at)`, user); err != nil {
			return err
		}
	}
	logger.Info("users data inserted successfully", logrus.Fields{constants.LoggerCategory: constants.LoggerCategorySeeder})

	return
}
