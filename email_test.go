package main

import (
	"github.com/magiconair/properties/assert"
	"github.com/stretchr/testify/require"
)

func (s *Suite) TestExample() {
	var err error
	emails := []string{
		"fuziontech@gmail.com",
		"test@test.com",
		"bob@builder.com",
	}
	for _, email := range emails {
		err = saveEmail(s.ctx, email)
		require.NoError(s.T(), err)
	}
	dbe, err := getAllEmails(s.ctx)
	require.NoError(s.T(), err)

	var dbEmails []string
	for _, e := range dbe {
		dbEmails = append(dbEmails, e.Email)
	}

	assert.Equal(s.T(), dbEmails, emails)
}
