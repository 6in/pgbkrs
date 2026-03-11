package foreignkey

import (
	"fmt"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts ForeignKeyDef to/from YAML bytes.
type Serializer struct{}

// Serialize converts an ObjectDef to YAML bytes.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

// Deserialize converts YAML bytes to an ObjectDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	return nil, fmt.Errorf("not implemented")
}
