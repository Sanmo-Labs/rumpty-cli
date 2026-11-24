package db_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/api/mocks"
	"github.com/cloverworks-hq/rumpty-cli/internal/app"
	"github.com/cloverworks-hq/rumpty-cli/internal/config"
	"github.com/cloverworks-hq/rumpty-cli/internal/db"
	"github.com/cloverworks-hq/rumpty-cli/internal/term"
)

func testDatabase() api.Database {
	return api.Database{
		UID:        "db-1",
		Name:       "Production",
		Slug:       "prod-db",
		Engine:     "postgres",
		Status:     "ready",
		ZoneSlug:   "lag-1",
		StorageGiB: 10,
		VCPU:       1,
		MemoryMiB:  1024,
	}
}

func newDeleteRuntime(t *testing.T, mock *mocks.MockClientAPI, in io.Reader) (*app.Runtime, *bytes.Buffer) {
	t.Helper()
	out := &bytes.Buffer{}
	return &app.Runtime{
		Config:    &config.Config{Token: "tok", Workspace: "prod-team"},
		Streams:   term.Streams{In: in, Out: out, ErrOut: out},
		APIClient: mock,
	}, out
}

func TestDelete_refusesWithoutConfirmationWhenNotInteractive(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListDatabases(gomock.Any(), "prod-team").Return([]api.Database{testDatabase()}, nil)

	rt, _ := newDeleteRuntime(t, mock, strings.NewReader(""))

	err := db.Delete(context.Background(), rt, "prod-db", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--yes")
}

func TestDelete_withYesDeletesAndWaits(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListDatabases(gomock.Any(), "prod-team").Return([]api.Database{testDatabase()}, nil)
	mock.EXPECT().DeleteDatabase(gomock.Any(), "prod-team", "db-1", gomock.Any()).Return(api.DatabaseOperationResult{
		OperationID: "op-1",
		Database:    testDatabase(),
	}, nil)
	mock.EXPECT().GetOperation(gomock.Any(), "prod-team", "op-1").Return(api.Operation{
		UID:    "op-1",
		Status: "succeeded",
	}, nil)

	rt, out := newDeleteRuntime(t, mock, strings.NewReader(""))

	require.NoError(t, db.Delete(context.Background(), rt, "prod-db", true))
	assert.Contains(t, out.String(), "Deleted prod-db")
}

func TestDelete_notFound(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	mock.EXPECT().ListDatabases(gomock.Any(), "prod-team").Return([]api.Database{}, nil)

	rt, _ := newDeleteRuntime(t, mock, strings.NewReader(""))

	err := db.Delete(context.Background(), rt, "missing-db", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestDelete_ambiguousRef(t *testing.T) {
	ctrl := gomock.NewController(t)
	mock := mocks.NewMockClientAPI(ctrl)
	first, second := testDatabase(), testDatabase()
	second.UID, second.Slug = "db-2", "staging-db"
	mock.EXPECT().ListDatabases(gomock.Any(), "prod-team").Return([]api.Database{first, second}, nil)

	rt, _ := newDeleteRuntime(t, mock, strings.NewReader(""))

	err := db.Delete(context.Background(), rt, "Production", true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple databases match")
}
