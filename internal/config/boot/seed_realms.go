package boot

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/monkeydioude/goauth/v2/internal/domain/entities"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SeedRealm struct {
	Name         string
	Kind         string
	AllowNewUser bool
}

// SeedRealmsBoot reads SEED_REALMS as comma-separated name:kind[:signup] entries.
func SeedRealmsBoot() ([]SeedRealm, error) {
	value := os.Getenv("SEED_REALMS")
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	var realms []SeedRealm
	for _, entry := range strings.Split(value, ",") {
		fields := strings.Split(entry, ":")
		if len(fields) < 2 || len(fields) > 3 {
			return nil, fmt.Errorf("SEED_REALMS entry %q: expected name:kind[:signup]", entry)
		}
		realm := SeedRealm{Name: strings.TrimSpace(fields[0]), Kind: strings.TrimSpace(fields[1])}
		if realm.Name == "" {
			return nil, fmt.Errorf("SEED_REALMS entry %q: name cannot be empty", entry)
		}
		if realm.Kind != entities.RealmKindHuman && realm.Kind != entities.RealmKindService {
			return nil, fmt.Errorf("SEED_REALMS entry %q: unknown kind %q; expected human or service", entry, realm.Kind)
		}
		if len(fields) == 3 {
			if strings.TrimSpace(fields[2]) != "signup" {
				return nil, fmt.Errorf("SEED_REALMS entry %q: unknown flag %q; expected signup", entry, fields[2])
			}
			if realm.Kind == entities.RealmKindService {
				return nil, fmt.Errorf("SEED_REALMS entry %q: service realms cannot allow signup", entry)
			}
			realm.AllowNewUser = true
		}
		realms = append(realms, realm)
	}
	return realms, nil
}

// seedRealms only creates missing realms. The name constraint includes deleted
// realms, so a conflict must be read unscoped before checking the stored kind.
func seedRealms(db *gorm.DB, realms []SeedRealm) error {
	for _, seed := range realms {
		realm := entities.Realm{Name: seed.Name, Kind: seed.Kind, AllowNewUser: seed.AllowNewUser}
		created := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}},
			DoNothing: true,
		}).Create(&realm)
		if created.Error != nil {
			return fmt.Errorf("SEED_REALMS: create realm %q: %w", seed.Name, created.Error)
		}

		// Read after the insert even on conflict: another instance may have
		// created the realm while this one was starting.
		var stored entities.Realm
		if err := db.Unscoped().Where("name = ?", seed.Name).First(&stored).Error; err != nil {
			return fmt.Errorf("SEED_REALMS: read realm %q: %w", seed.Name, err)
		}
		if stored.DeletedAt.Valid {
			slog.Warn("skipping soft-deleted realm; remove it from SEED_REALMS", "realm", seed.Name)
			continue
		}
		if stored.Kind != seed.Kind {
			return fmt.Errorf("SEED_REALMS: realm %q has kind %q, configured kind is %q", seed.Name, stored.Kind, seed.Kind)
		}
		if created.RowsAffected > 0 {
			slog.Info("created realm from SEED_REALMS", "realm", seed.Name, "kind", seed.Kind)
		}
	}
	return nil
}
