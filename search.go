/**
  bloqsenjin - An interface that gives access to the graph database that will
  be used for a search engine on a Bloqs marketplace.
  Copyright (C) 2023  João Torres

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU Affero General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU Affero General Public License for more details.

  You should have received a copy of the GNU Affero General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package enjin

import (
	"strings"

	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j/dbtype"
)

var (
	preferences set[*Preference]
)

type Preference string

func (proxy DriverProxy) GetPreferences(useCache bool) ([]*Preference, error) {
	if !preferences.isEmpty() && useCache {
		return *preferences.enumerate(), nil
	}

	session := proxy.createSession(neo4j.AccessModeRead)

	defer session.Close(proxy.ctx)

	cypher, params := "MATCH (p:Preference) RETURN p", make(map[string]any)

	res, err := session.Run(proxy.ctx, cypher, params)

	if err != nil {
		return make([]*Preference, 0), err
	}

	records, err := res.Collect(proxy.ctx)

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

	defer session.Close(proxy.ctx)

	cypher, params := "MERGE (:Preference {name: $p})", make(map[string]any)
	params["p"] = strings.ToLower(string(preference))

	_, err := session.ExecuteWrite(proxy.ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		return tx.Run(proxy.ctx, cypher, params)
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

//func main() {
//	conf, err := CreateConfig("credentials.json")
//
//	if err != nil {
//		panic(err)
//	}
//
//	auth := neo4j.BasicAuth(conf.User, conf.Password, "")
//
//	driver, err := neo4j.NewDriverWithContext(conf.GetDbUri(), auth)
//
//	if err != nil {
//		panic(err)
//	}
//
//	ctx := context.Background()
//
//	defer driver.Close(ctx)
//
//	proxy := DriverProxy{ctx, driver, "neo4j"}
//
//	proxy.GetPreferences()
//
//	value, err := proxy.GetPreferences()
//
//	if err != nil {
//		panic(err)
//	}
//
//	fmt.Println(*value[0])
//}
