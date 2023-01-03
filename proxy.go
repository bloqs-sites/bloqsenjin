package enjin

import (
	"context"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
)

type DriverProxy struct {
	ctx    context.Context
	driver neo4j.DriverWithContext
	db     string
}

func (p DriverProxy) createSession(mode neo4j.AccessMode) neo4j.SessionWithContext {
	return p.driver.NewSession(p.ctx, neo4j.SessionConfig{
		AccessMode:   mode,
		DatabaseName: p.db,
	})
}
