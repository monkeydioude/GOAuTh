package boot

import (
	"os"
	"testing"

	"github.com/monkeydioude/goauth/v2/internal/config/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedRealmsBoot(t *testing.T) {
	for _, tt := range []struct {
		name  string
		value string
		want  []SeedRealm
	}{
		{name: "empty"},
		{name: "whitespace", value: " \t "},
		{name: "human without signup", value: "a:human", want: []SeedRealm{{Name: "a", Kind: "human"}}},
		{name: "both kinds", value: "a:human:signup,b:service", want: []SeedRealm{
			{Name: "a", Kind: "human", AllowNewUser: true},
			{Name: "b", Kind: "service"},
		}},
		{name: "surrounding whitespace", value: " a : human : signup , b : service ", want: []SeedRealm{
			{Name: "a", Kind: "human", AllowNewUser: true},
			{Name: "b", Kind: "service"},
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SEED_REALMS", tt.value)
			got, err := SeedRealmsBoot()
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSeedRealmsBootUnsetDoesNothing(t *testing.T) {
	t.Setenv("SEED_REALMS", "")
	require.NoError(t, os.Unsetenv("SEED_REALMS"))
	realms, err := SeedRealmsBoot()
	require.NoError(t, err)
	assert.Empty(t, realms)
	// Empty configuration must not even access the database.
	assert.NoError(t, seedRealms(nil, realms))
}

func TestSeedRealmsBootRefusesMalformedEntries(t *testing.T) {
	for _, entry := range []string{
		"b:service:signup", "c:robot", ":human", " :human", "a", "a:",
		"a:human:unknown", "a:human:", "a:human:signup:extra", "",
	} {
		t.Run(entry, func(t *testing.T) {
			// Validate the whole configuration before any realm can be seeded.
			t.Setenv("SEED_REALMS", "valid:human,"+entry)
			realms, err := SeedRealmsBoot()
			require.Error(t, err)
			assert.ErrorContains(t, err, "SEED_REALMS")
			assert.ErrorContains(t, err, entry)
			assert.Nil(t, realms)
		})
	}
}

func TestLayoutBootRefusesInvalidSeedRealmsBeforeConnecting(t *testing.T) {
	t.Setenv(consts.TRUSTED_PROXIES, "")
	t.Setenv("SEED_REALMS", "b:service:signup")
	t.Setenv(consts.DB_PATH, "invalid database connection")

	res := LayoutBoot(nil, nil, nil)

	require.True(t, res.IsErr())
	assert.ErrorContains(t, res.Error, "b:service:signup")
}
