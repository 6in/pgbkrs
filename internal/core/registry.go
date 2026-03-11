package core

// CommandRegistry maps ObjectKind to the triple of (SchemaFetcher, Serializer, DDLGenerator).
// A single Register() call makes a new ObjectKind resolvable via Fetcher, Serializer, and Generator.
type CommandRegistry struct {
	fetchers    map[ObjectKind]SchemaFetcher
	serializers map[ObjectKind]Serializer
	generators  map[ObjectKind]DDLGenerator
}

// NewCommandRegistry returns a registry with all three maps initialized (non-nil).
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		fetchers:    make(map[ObjectKind]SchemaFetcher),
		serializers: make(map[ObjectKind]Serializer),
		generators:  make(map[ObjectKind]DDLGenerator),
	}
}

// Register stores the SchemaFetcher, Serializer, and DDLGenerator for the given ObjectKind.
// A single call is sufficient to make all three components resolvable.
func (r *CommandRegistry) Register(kind ObjectKind, f SchemaFetcher, s Serializer, g DDLGenerator) {
	r.fetchers[kind] = f
	r.serializers[kind] = s
	r.generators[kind] = g
}

// Fetcher returns the SchemaFetcher registered for kind, or nil if not registered.
func (r *CommandRegistry) Fetcher(kind ObjectKind) SchemaFetcher { return r.fetchers[kind] }

// Serializer returns the Serializer registered for kind, or nil if not registered.
func (r *CommandRegistry) Serializer(kind ObjectKind) Serializer { return r.serializers[kind] }

// Generator returns the DDLGenerator registered for kind, or nil if not registered.
func (r *CommandRegistry) Generator(kind ObjectKind) DDLGenerator { return r.generators[kind] }
