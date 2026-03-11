package table

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts TableDef to/from YAML bytes.
type Serializer struct{}

// Serialize converts a TableDef ObjectDef to YAML bytes matching spec section 5.2.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Deserialize converts YAML bytes to a *core.TableDef ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	return nil, fmt.Errorf("not implemented")
}
