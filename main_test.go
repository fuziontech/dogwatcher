package main

import (
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"testing"
)

type Suite struct {
	suite.Suite
	ctx   ServerContext
	email *Email
}

func (s *Suite) SetupSuite() {
	var err error
	const postgresURL = "postgresql://james:james@localhost/test_dogwatcher"

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	}
	s.ctx.gdb, err = gorm.Open(postgres.Open(postgresURL), gormConfig)
	require.NoError(s.T(), err)

	err = autoMigrate(s.ctx)
	require.NoError(s.T(), err)
}

func (s *Suite) SetupTest() {
	s.ctx.gdb.Exec("TRUNCATE TABLE emails")
	s.ctx.gdb.Exec("TRUNCATE TABLE doggos")
}

func TestSuite(t *testing.T) {
	suite.Run(t, new(Suite))
}
