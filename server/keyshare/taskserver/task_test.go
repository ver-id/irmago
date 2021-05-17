//+build !local_tests

package taskserver

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	irma "github.com/privacybydesign/irmago"
	"github.com/privacybydesign/irmago/internal/test"
	"github.com/privacybydesign/irmago/server/keyshare"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func init() {
	irma.Logger.SetLevel(logrus.FatalLevel)
}

func TestCleanupEmails(t *testing.T) {
	SetupDatabase(t)
	defer TeardownDatabase(t)

	db, err := sql.Open("pgx", test.PostgresTestUrl)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO irma.users (id, username, last_seen, language, coredata, pin_counter, pin_block_date) VALUES (15, 'testuser', 15, '', '', 0,0)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO irma.emails (user_id, email, delete_on) VALUES (15, 'test@test.com', NULL), (15, 'test2@test.com', $1), (15, 'test3@test.com', 0)", time.Now().Add(time.Hour).Unix())
	require.NoError(t, err)

	th, err := newHandler(&Configuration{DBConnstring: test.PostgresTestUrl, Logger: irma.Logger})
	require.NoError(t, err)

	th.cleanupEmails()

	res, err := db.Query("SELECT COUNT(*) FROM irma.emails")
	require.NoError(t, err)
	require.True(t, res.Next())
	var count int
	require.NoError(t, res.Scan(&count))
	assert.Equal(t, 2, count)
	_ = res.Close()
}

func TestCleanupTokens(t *testing.T) {
	SetupDatabase(t)
	defer TeardownDatabase(t)

	db, err := sql.Open("pgx", test.PostgresTestUrl)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO irma.users (id, username, last_seen, language, coredata, pin_counter, pin_block_date) VALUES (15, 'testuser', 15, '', '', 0,0)")
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO irma.email_verification_tokens (token, user_id, email, expiry) VALUES ('t1', 15, 't1@test.com', 0), ('t2', 15, 't2@test.com', $1)", time.Now().Add(time.Hour).Unix())
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO irma.email_login_tokens (token, email, expiry) VALUES ('t1', 't1@test.com', 0), ('t2', 't2@test.com', $1)", time.Now().Add(time.Hour).Unix())
	require.NoError(t, err)

	th, err := newHandler(&Configuration{DBConnstring: test.PostgresTestUrl, Logger: irma.Logger})
	require.NoError(t, err)

	th.cleanupTokens()

	res, err := db.Query("SELECT COUNT(*) FROM irma.email_verification_tokens")
	require.NoError(t, err)
	require.True(t, res.Next())
	var count int
	require.NoError(t, res.Scan(&count))
	assert.Equal(t, 1, count)
	_ = res.Close()

	res, err = db.Query("SELECT COUNT(*) FROM irma.email_login_tokens")
	require.NoError(t, err)
	require.True(t, res.Next())
	require.NoError(t, res.Scan(&count))
	assert.Equal(t, 1, count)
	_ = res.Close()
}

func TestCleanupAccounts(t *testing.T) {
	SetupDatabase(t)
	defer TeardownDatabase(t)

	db, err := sql.Open("pgx", test.PostgresTestUrl)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO irma.users (id, username, language, coredata, pin_counter, pin_block_date, last_seen, delete_on) VALUES (15, 'testuser', '', '', 0,0, 0, NULL), (16, 't2', '', '', 0, 0, 0, $1-3600), (17, 't3', '', '', 0, 0, $1, $1-3600), (18, 't4', '', NULL, 0, 0, $1, $1-3600)", time.Now().Unix())
	require.NoError(t, err)

	th, err := newHandler(&Configuration{DBConnstring: test.PostgresTestUrl, Logger: irma.Logger})
	require.NoError(t, err)

	th.cleanupAccounts()

	res, err := db.Query("SELECT COUNT(*) FROM irma.users")
	require.NoError(t, err)
	require.True(t, res.Next())
	var count int
	require.NoError(t, res.Scan(&count))
	assert.Equal(t, 2, count)
	_ = res.Close()
}

func TestExpireAccounts(t *testing.T) {
	testdataPath := test.FindTestdataFolder(t)
	SetupDatabase(t)
	defer TeardownDatabase(t)

	db, err := sql.Open("pgx", test.PostgresTestUrl)
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO irma.users(id, username, language, coredata, pin_counter, pin_block_date, last_seen) VALUES (15, 'A', '', '', 0, 0, $1-12*3600), (16, 'B', '', '', 0, 0, 0), (17, 'C', '', '', 0, 0, 0)", time.Now().Unix())
	require.NoError(t, err)
	_, err = db.Exec("INSERT INTO irma.emails (user_id, email, delete_on) VALUES (15, 'test@test.com', NULL), (16, 'test@test.com', NULL)")
	require.NoError(t, err)

	th, err := newHandler(&Configuration{
		DBConnstring: test.PostgresTestUrl,
		DeleteDelay:  30,
		ExpiryDelay:  1,
		EmailConfiguration: keyshare.EmailConfiguration{
			EmailServer:     "localhost:1025",
			EmailFrom:       "test@test.com",
			DefaultLanguage: "en",
		},
		DeleteExpiredAccountFiles: map[string]string{
			"en": filepath.Join(testdataPath, "emailtemplate.html"),
		},
		DeleteExpiredAccountSubject: map[string]string{
			"en": "testsubject",
		},
		Logger: irma.Logger,
	})
	require.NoError(t, err)

	th.expireAccounts()

	res, err := db.Query("SELECT COUNT(*) FROM irma.users WHERE delete_on IS NOT NULL")
	require.NoError(t, err)
	require.True(t, res.Next())
	var count int
	require.NoError(t, res.Scan(&count))
	assert.Equal(t, 1, count)
	_ = res.Close()
}

func SetupDatabase(t *testing.T) {
	test.RunScriptOnDB(t, "../keyshareserver/cleanup.sql", true)
	test.RunScriptOnDB(t, "../keyshareserver/schema.sql", false)
}

func TeardownDatabase(t *testing.T) {
	test.RunScriptOnDB(t, "../keyshareserver/cleanup.sql", false)
}
