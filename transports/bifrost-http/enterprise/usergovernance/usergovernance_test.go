package usergovernance

import (
	"testing"

	schemas "github.com/maximhq/bifrost/core/schemas"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockLogger struct{}

func (m *mockLogger) SetLevel(level schemas.LogLevel)                        {}
func (m *mockLogger) SetOutputType(outputType schemas.LoggerOutputType)      {}
func (m *mockLogger) Debug(msg string, args ...any)                          {}
func (m *mockLogger) Info(msg string, args ...any)                           {}
func (m *mockLogger) Warn(msg string, args ...any)                          {}
func (m *mockLogger) Error(msg string, args ...any)                          {}
func (m *mockLogger) Fatal(msg string, args ...any)                          {}
func (m *mockLogger) LogHTTPRequest(level schemas.LogLevel, msg string) schemas.LogEventBuilder {
	return schemas.NoopLogEvent
}

func TestInit_NilConfig(t *testing.T) {
	p := Init(nil, &mockLogger{})
	require.NotNil(t, p)
	assert.Equal(t, PluginName, p.GetName())
	assert.Nil(t, p.Cleanup())
	assert.False(t, p.IsEnabled())
}

func TestInit_WithConfig(t *testing.T) {
	p := Init(&Config{Enabled: true}, &mockLogger{})
	require.NotNil(t, p)
	assert.True(t, p.IsEnabled())
}

func TestUserCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	user := &User{Name: "alice", Email: "alice@example.com"}
	err := p.CreateUser(user)
	require.NoError(t, err)
	assert.NotEmpty(t, user.ID)
	assert.True(t, user.Active)

	got, ok := p.GetUser(user.ID)
	require.True(t, ok)
	assert.Equal(t, "alice", got.Name)
}

func TestGetUser_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	_, ok := p.GetUser("nonexistent")
	assert.False(t, ok)
}

func TestUpdateUser(t *testing.T) {
	p := Init(nil, &mockLogger{})
	user := &User{Name: "alice"}
	p.CreateUser(user)

	budget := 100.0
	rpm := 60
	err := p.UpdateUser(user.ID, map[string]any{
		"name":          "Alice Smith",
		"email":        "alice.smith@example.com",
		"budget_limit": budget,
		"rate_limit_rpm": rpm,
		"active":       false,
	})
	require.NoError(t, err)

	got, _ := p.GetUser(user.ID)
	assert.Equal(t, "Alice Smith", got.Name)
	assert.Equal(t, "alice.smith@example.com", got.Email)
	assert.False(t, got.Active)
}

func TestUpdateUser_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.UpdateUser("nonexistent", map[string]any{"name": "x"})
	assert.Error(t, err)
}

func TestDeleteUser(t *testing.T) {
	p := Init(nil, &mockLogger{})
	user := &User{Name: "alice"}
	p.CreateUser(user)

	err := p.DeleteUser(user.ID)
	require.NoError(t, err)
	_, ok := p.GetUser(user.ID)
	assert.False(t, ok)
}

func TestDeleteUser_NotFound(t *testing.T) {
	p := Init(nil, &mockLogger{})
	err := p.DeleteUser("nonexistent")
	assert.Error(t, err)
}

func TestListUsers_FilterByTeam(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.CreateUser(&User{Name: "alice", TeamID: "team-1"})
	p.CreateUser(&User{Name: "bob", TeamID: "team-2"})
	p.CreateUser(&User{Name: "carol", TeamID: "team-1"})

	team1 := p.ListUsers("team-1", 0, 10)
	assert.Len(t, team1, 2)

	all := p.ListUsers("", 0, 0)
	assert.Len(t, all, 3)
}

func TestTeamCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	team := &Team{Name: "platform", LeaderID: "user-1"}
	err := p.CreateTeam(team)
	require.NoError(t, err)
	assert.NotEmpty(t, team.ID)

	got, ok := p.GetTeam(team.ID)
	require.True(t, ok)
	assert.Equal(t, "platform", got.Name)

	teams := p.ListTeams()
	assert.Len(t, teams, 1)
}

func TestBusinessUnitCRUD(t *testing.T) {
	p := Init(nil, &mockLogger{})

	bu := &BusinessUnit{Name: "engineering", TeamIDs: []string{"team-1"}}
	err := p.CreateBusinessUnit(bu)
	require.NoError(t, err)
	assert.NotEmpty(t, bu.ID)

	got, ok := p.GetBusinessUnit(bu.ID)
	require.True(t, ok)
	assert.Equal(t, "engineering", got.Name)

	bus := p.ListBusinessUnits()
	assert.Len(t, bus, 1)
}

func TestRecordUsageAndCheckBudget(t *testing.T) {
	p := Init(nil, &mockLogger{})
	budget := 50.0
	user := &User{Name: "alice", BudgetLimit: &budget}
	p.CreateUser(user)

	p.RecordUsage(user.ID, 10.0)
	p.RecordUsage(user.ID, 20.0)
	assert.True(t, p.CheckBudget(user.ID))

	p.RecordUsage(user.ID, 25.0)
	assert.False(t, p.CheckBudget(user.ID))
}

func TestCheckBudget_NoLimit(t *testing.T) {
	p := Init(nil, &mockLogger{})
	user := &User{Name: "alice"}
	p.CreateUser(user)
	assert.True(t, p.CheckBudget(user.ID))
}

func TestCheckBudget_UnknownUser(t *testing.T) {
	p := Init(nil, &mockLogger{})
	assert.True(t, p.CheckBudget("nonexistent"))
}

func TestCheckRateLimit(t *testing.T) {
	p := Init(nil, &mockLogger{})
	rpm := 100
	user := &User{Name: "alice", RateLimitRPM: &rpm}
	p.CreateUser(user)

	assert.True(t, p.CheckRateLimit(user.ID, 50))
	assert.False(t, p.CheckRateLimit(user.ID, 200))
}

func TestGetUserRankings(t *testing.T) {
	p := Init(nil, &mockLogger{})
	p.CreateUser(&User{Name: "alice"})
	p.CreateUser(&User{Name: "bob"})

	p.RecordUsage("alice-generated-id", 100.0)
	p.RecordUsage("bob-generated-id", 200.0)

	rankings := p.GetUserRankings(10)
	assert.NotEmpty(t, rankings)
	if len(rankings) >= 2 {
		assert.Equal(t, 1, rankings[0].Rank)
	}
}
