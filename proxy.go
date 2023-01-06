package enjin

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

var (
	preferences set[Preference]
)

func init() {
	preferences = buildSet[Preference]()
}

// DriverProxy works as an API to the operations to do with the database
type DriverProxy struct {
	ctx *context.Context         // context in case of need to cancel an operation
	drv *neo4j.DriverWithContext // refrence to the actual service
	db  *string                  // database that this proxy uses
}

func (px DriverProxy) createSession(mode neo4j.AccessMode) neo4j.SessionWithContext {
	drv := *px.drv
	return drv.NewSession(*px.ctx, neo4j.SessionConfig{
		AccessMode:   mode,
		DatabaseName: *px.db,
	})
}

func (px *DriverProxy) Close() {
	drv := *px.drv
	drv.Close(*px.ctx)
}

func (px DriverProxy) GetPreferences(useCache bool) ([]Preference, error) {
	if !preferences.isEmpty() && useCache {
		return nil, nil
	}

	session := px.createSession(neo4j.AccessModeRead)

	defer session.Close(*px.ctx)

	cypher := "MATCH (p:Preference) RETURN p"

	res, err := session.Run(*px.ctx, cypher, nil)

	if err != nil {
		return nil, err
	}

	records, err := res.Collect(*px.ctx)

	if err != nil {
		return nil, err
	}

	for _, record := range records {
		node, exists := record.Get("p")

		if exists {
			node, ok := node.(dbtype.Node)

			if ok {
				name := Preference(strings.ToLower(node.Props["name"].(string)))
				preferences.add(name)
			}
		}
	}

	return preferences.enumerate(), err
}

func (px DriverProxy) NewPreference(preference Preference) error {
	p := strings.ToLower(string(preference))
	session := px.createSession(neo4j.AccessModeWrite)

	defer session.Close(*px.ctx)

	cypher, params := "MERGE (:Preference {name: $p})", make(map[string]any)
	params["p"] = p

	_, err := session.ExecuteWrite(*px.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(*px.ctx, cypher, params)
	})

	//cypher = `MATCH (a:Preference {name: $p}), (b:Preference)
	//WHERE NOT a = b
	//MERGE (a)-[:SHARES {value: 0}]->(b)`

	//_, err = session.ExecuteWrite(proxy.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
	//	return tx.Run(proxy.ctx, cypher, params)
	//})

	if err == nil {
		preferences.add(Preference(p))
	}

	return err
}

func (px DriverProxy) createGlobal() error {
	session := px.createSession(neo4j.AccessModeWrite)

	defer session.Close(*px.ctx)

	cypher, params := "MERGE (:Global)", make(map[string]any)

	_, err := session.ExecuteWrite(*px.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(*px.ctx, cypher, params)
	})

	return err
}

func (px DriverProxy) CreateClient(id string, likes []Preference) error {
	session := px.createSession(neo4j.AccessModeWrite)

	defer session.Close(*px.ctx)

	json, err := json.Marshal(likes)

	if err != nil {
		return nil
	}

	cypher, params := fmt.Sprintf(`WITH %s as likes
    MATCH (u:Client {id: $id})
    SET u.lvl = 1
    FOREACH (l in likes
        | MERGE (u)-[:LIKES {weight: $w}]->(:Preference {name: l}))`, json), make(map[string]any)
	params["id"] = id
	params["w"] = math.Floor(float64(100 / len(likes)))

	_, err = session.ExecuteWrite(*px.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(*px.ctx, cypher, params)
	})

	if err != nil {
		return nil
	}

	cypher = `MATCH (l1:Preference {name: $lf}), (l2:Preference {name: $ls})
    MERGE (l1)-[s:SHARES]-(l2)
    SET s.weight = 1`
	for i, p := range likes {
		params["lf"] = p
		for j := i + 1; j < len(likes); j++ {
			params["ls"] = likes[j]

			_, err = session.ExecuteWrite(*px.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
				return tx.Run(*px.ctx, cypher, params)
			})

			if err != nil {
				return nil
			}
		}
	}

	return err
}

func (px DriverProxy) InitiateDatabase(preferences []Preference) {
	for _, p := range preferences {
		go px.NewPreference(p)
	}

	go px.createGlobal()
}
