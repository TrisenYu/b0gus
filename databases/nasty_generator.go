package databases

import (
	"fmt"
	"strings"
)

/*
	Automatically generate structures for interacting with different database backends
*/

func topologySort() {

}

/*
{
	"name": {}
}
*/

func Parse(payloads map[string]map[string][]any) error {
	for tab_name, table := range payloads {
		// TODO: this is only a function used for relation table but not for NoSQL
		curr_definition := fmt.Sprintf("type %s struct {", tab_name)
		for col_name, col_type := range table {
			curr_definition += "\t" + col_name // then assign its type and definitions of tag
			for _, val := range col_type {
				switch val.(type) {
				case map[string]any:
					// type or foreign reference or constrain<map[string]any, also the last one>
				case string:
					switch strings.Trim(strings.ToLower(val.(string)), " ") {
					case "primarykey":
					case "autoincrement":
					case "notnull":
					case "unique":
					case "index":
					default:
						return fmt.Errorf(
							"unrecognized field was found when parsing the database meta-payload:%v", val,
						)
					}
				default:
					return fmt.Errorf(
						"unexpected field<%v> was found in %s:%s",
						val, tab_name, col_name,
					)
				}
			}
		}
		curr_definition += fmt.Sprintf("\n}\n")
		// name
		// Then we can generate one structure as a table here, but since we have restrictions,
		// we still need to generate tag for them
	}
	return nil
	/*

	 */
}

//go:generate
/*
	It seems that we human can not define these field in an advanced and elegant way
	but only list them and exhaust.
*/
var (
	_database_types = Parse(map[string]map[string][]any{
		// First table
		"AddrInfo": map[string][]any{
			"IP": []any{
				map[string]any{
					"type":    "uint64",
					"index":   "ip_name_idx",
					"comment": "Attackers' IP address",
				},
				"not null", "unique",
			},
			"AttemptTime": []any{
				map[string]any{
					"default": 0,
					"type":    "uint64",
				},
			},
		},

		// Second table
		"PortInfo": map[string][]any{
			"port": []any{
				map[string]any{
					"type": "uint16",
				},
			},
			"addr_id": []any{
				map[string]any{
					"refer": []string{"AddrInfo", "IP"},
					"constraint": map[string]any{
						"onUpdate": "CASCADE",
						"onDelete": "DELETE",
					},
					"type": "uint64",
				},
			},
			"last_time": []any{
				map[string]any{
					"autoUpdateTime": "nano",
					"comment":        "last time that attacker lanuched an attack",
					"type":           "int64",
				},
			},
		},

		"SSHversionInfo": map[string][]any{
			"client_version": []any{
				"unique", "not null",
				map[string]any{
					"type": "string",
				},
			},
			"FirstRecordTime": []any{
				map[string]any{
					"autoCreateTime": "nano",
					"type":           "int64",
				},
			},
			"LastTime": []any{
				map[string]any{
					"autoCreateTime": "nano",
					"type":           "int64",
				},
			},
		},
	})
)
