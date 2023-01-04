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

// Package enjin gives an interface to maniputade nodes and relationships in a
// graph database (neo4j) so that you can use that information in a search
// engine of an Bloqs marketplace website.
package enjin

// A Preference it's a representation of an Preference node on the database
type Preference string

//func main() {
//	conf, err := CreateConfig("credentials.json")
//
//	if err != nil {
//		panic(err)
//	}
//
//	proxy := conf.CreateProxy()
//
//	value, err := proxy.GetPreferences(false)
//
//	if err != nil {
//		panic(err)
//	}
//
//	fmt.Println(*value[0])
//}
