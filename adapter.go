package goten

import adp "github.com/dnahilman/goten/adapter"

// Type aliases so callers can use goten.Adapter, goten.Query, etc.
type Adapter = adp.Adapter
type Query = adp.Query
type Where = adp.Where

// EQ constructs an equality Where clause.
var EQ = adp.EQ
