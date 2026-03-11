package table

import (
	"fmt"

	"go.yaml.in/yaml/v3"

	"github.com/pgbkrs/pgbackup/internal/core"
)

// Compile-time interface check.
var _ core.Serializer = (*Serializer)(nil)

// Serializer converts TableDef to/from YAML bytes matching spec section 5.2.
type Serializer struct{}

// Intermediate YAML structs matching spec section 5.2 exactly.
// These decouple the YAML format from internal core types.

type yamlTableDef struct {
	Kind         string            `yaml:"kind"`
	Schema       string            `yaml:"schema"`
	Name         string            `yaml:"name"`
	Columns      []yamlColumn      `yaml:"columns"`
	Constraints  yamlConstraints   `yaml:"constraints"`
	Indexes      []yamlIndex       `yaml:"indexes,omitempty"`
	Partitioning *yamlPartitioning `yaml:"partitioning"`
	RLS          yamlRLS           `yaml:"rls"`
	Data         *yamlData         `yaml:"data,omitempty"`
}

type yamlColumn struct {
	Name     string `yaml:"name"`
	Type     string `yaml:"type"`
	Nullable bool   `yaml:"nullable"` // NO omitempty: false is meaningful
	Default  string `yaml:"default,omitempty"`
}

type yamlConstraints struct {
	PrimaryKey *yamlPK      `yaml:"primary_key,omitempty"`
	Unique     []yamlUnique `yaml:"unique,omitempty"`
	Check      []yamlCheck  `yaml:"check,omitempty"`
}

type yamlPK struct {
	Name    string   `yaml:"name"`
	Columns []string `yaml:"columns,flow"`
}

type yamlUnique struct {
	Name       string `yaml:"name"`
	Definition string `yaml:"definition"`
}

type yamlCheck struct {
	Name       string `yaml:"name"`
	Definition string `yaml:"definition"`
}

type yamlIndex struct {
	Name   string `yaml:"name"`
	Method string `yaml:"method"`
}

type yamlPartitioning struct {
	Strategy      string   `yaml:"strategy"`
	KeyExpression string   `yaml:"key_expression"`
	Children      []string `yaml:"children"`
}

type yamlRLS struct {
	Enabled bool `yaml:"enabled"`
}

type yamlData struct {
	File     string   `yaml:"file"`
	Columns  []string `yaml:"columns"`
	RowCount int64    `yaml:"row_count"`
	Checksum string   `yaml:"checksum"`
}

// toYAML converts a core.TableDef to the intermediate YAML struct.
func toYAML(td *core.TableDef) yamlTableDef {
	yd := yamlTableDef{
		Kind:   string(td.Kind),
		Schema: td.Schema,
		Name:   td.Name,
		RLS:    yamlRLS{Enabled: td.RLS.Enabled},
	}

	// Columns
	for _, col := range td.Columns {
		yd.Columns = append(yd.Columns, yamlColumn{
			Name:     col.Name,
			Type:     col.Type,
			Nullable: col.Nullable,
			Default:  col.Default,
		})
	}

	// Constraints
	if td.Constraints.PrimaryKey != nil {
		yd.Constraints.PrimaryKey = &yamlPK{
			Name:    td.Constraints.PrimaryKey.Name,
			Columns: td.Constraints.PrimaryKey.Columns,
		}
	}
	for _, uc := range td.Constraints.Unique {
		yd.Constraints.Unique = append(yd.Constraints.Unique, yamlUnique{
			Name:       uc.Name,
			Definition: uc.Definition,
		})
	}
	for _, cc := range td.Constraints.Check {
		yd.Constraints.Check = append(yd.Constraints.Check, yamlCheck{
			Name:       cc.Name,
			Definition: cc.Definition,
		})
	}

	// Indexes
	for _, idx := range td.Indexes {
		yd.Indexes = append(yd.Indexes, yamlIndex{
			Name:   idx.Name,
			Method: idx.Method,
		})
	}

	// Partitioning
	if td.Partitioning != nil {
		yd.Partitioning = &yamlPartitioning{
			Strategy:      td.Partitioning.Strategy,
			KeyExpression: td.Partitioning.KeyExpression,
			Children:      td.Partitioning.Children,
		}
	}

	// DataMeta: emit data: block in def.yaml when CSV export metadata is available.
	if td.DataMeta != nil {
		yd.Data = &yamlData{
			File:     td.DataMeta.File,
			Columns:  td.DataMeta.Columns,
			RowCount: td.DataMeta.RowCount,
			Checksum: td.DataMeta.Checksum,
		}
	}

	return yd
}

// fromYAML converts the intermediate YAML struct back to a *core.TableDef.
func fromYAML(yd yamlTableDef) *core.TableDef {
	td := &core.TableDef{
		ObjectHeader: core.ObjectHeader{
			Kind:   core.ObjectKind(yd.Kind),
			Schema: yd.Schema,
			Name:   yd.Name,
		},
		RLS: core.RLSDef{Enabled: yd.RLS.Enabled},
	}

	// Columns
	for _, col := range yd.Columns {
		td.Columns = append(td.Columns, core.ColumnDef{
			Name:     col.Name,
			Type:     col.Type,
			Nullable: col.Nullable,
			Default:  col.Default,
		})
	}

	// Constraints
	if yd.Constraints.PrimaryKey != nil {
		td.Constraints.PrimaryKey = &core.PrimaryKeyDef{
			Name:    yd.Constraints.PrimaryKey.Name,
			Columns: yd.Constraints.PrimaryKey.Columns,
		}
	}
	for _, uc := range yd.Constraints.Unique {
		td.Constraints.Unique = append(td.Constraints.Unique, core.UniqueConstraintDef{
			Name:       uc.Name,
			Definition: uc.Definition,
		})
	}
	for _, cc := range yd.Constraints.Check {
		td.Constraints.Check = append(td.Constraints.Check, core.CheckConstraintDef{
			Name:       cc.Name,
			Definition: cc.Definition,
		})
	}

	// Indexes (method only; Definition not stored in YAML)
	for _, idx := range yd.Indexes {
		td.Indexes = append(td.Indexes, core.IndexDef{
			Name:   idx.Name,
			Method: idx.Method,
		})
	}

	// Partitioning
	if yd.Partitioning != nil {
		td.Partitioning = &core.PartitionDef{
			Strategy:      yd.Partitioning.Strategy,
			KeyExpression: yd.Partitioning.KeyExpression,
			Children:      yd.Partitioning.Children,
		}
	}

	return td
}

// Serialize converts a *core.TableDef to YAML bytes matching spec section 5.2.
func (s *Serializer) Serialize(def core.ObjectDef) ([]byte, error) {
	td, ok := def.(*core.TableDef)
	if !ok {
		return nil, fmt.Errorf("table.Serializer.Serialize: expected *core.TableDef, got %T", def)
	}

	yd := toYAML(td)
	return yaml.Marshal(&yd)
}

// Deserialize converts YAML bytes to a *core.TableDef.
func (s *Serializer) Deserialize(data []byte) (core.ObjectDef, error) {
	var yd yamlTableDef
	if err := yaml.Unmarshal(data, &yd); err != nil {
		return nil, fmt.Errorf("table.Serializer.Deserialize: %w", err)
	}

	return fromYAML(yd), nil
}
