package enjin

import (
	"context"

	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

var (
	preferences set[*Preference]
)

// DriverProxy works as an API to the operations to do with the database
type DriverProxy struct {
	ctx    *context.Context         // context in case of need to cancel an operation
	driver *neo4j.DriverWithContext // refrence to the actual service
	db     *string                  // database that this proxy uses
}

func (p DriverProxy) createSession(mode neo4j.AccessMode) neo4j.SessionWithContext {
	driver := *p.driver
	return driver.NewSession(*p.ctx, neo4j.SessionConfig{
		AccessMode:   mode,
		DatabaseName: *p.db,
	})
}

func (p *DriverProxy) Close() {
	driver := *p.driver
	driver.Close(*p.ctx)
}

func (proxy DriverProxy) GetPreferences(useCache bool) ([]*Preference, error) {
	if !preferences.isEmpty() && useCache {
		return *preferences.enumerate(), nil
	}

	session := proxy.createSession(neo4j.AccessModeRead)

	defer session.Close(*proxy.ctx)

	cypher, params := "MATCH (p:Preference) RETURN p", make(map[string]any)

	res, err := session.Run(*proxy.ctx, cypher, params)

	if err != nil {
		return make([]*Preference, 0), err
	}

	records, err := res.Collect(*proxy.ctx)

	if err != nil {
		return make([]*Preference, 0), err
	}

	for _, record := range records {
		node, exists := record.Get("p")

		if exists {
			node, ok := node.(dbtype.Node)

			if ok {
				name := Preference(node.Props["name"].(string))
				preferences.add(&name)
			}
		}
	}

	return *preferences.enumerate(), err
}

func (proxy DriverProxy) NewPreference(preference Preference) error {
	session := proxy.createSession(neo4j.AccessModeWrite)

	defer session.Close(*proxy.ctx)

	cypher, params := "MERGE (:Preference {name: $p})", make(map[string]any)
	params["p"] = strings.ToLower(string(preference))

	_, err := session.ExecuteWrite(*proxy.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(*proxy.ctx, cypher, params)
	})

	//cypher = `MATCH (a:Preference {name: $p}), (b:Preference)
	//WHERE NOT a = b
	//MERGE (a)-[:SHARES {value: 0}]->(b)`

	//_, err = session.ExecuteWrite(proxy.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
	//	return tx.Run(proxy.ctx, cypher, params)
	//})

	if err == nil {
		preferences.add(&preference)
	}

	return err
}
