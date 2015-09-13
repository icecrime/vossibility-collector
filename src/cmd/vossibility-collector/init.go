package main

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
	"github.com/mattbaird/elastigo/api"
	"github.com/mattbaird/elastigo/indices"
)

var initCommand = cli.Command{
	Name:   "init",
	Usage:  "initialized elasticsearch indices",
	Action: doInitCommand,
}

// TODO This depends on the mapping defined in the config, and should be store
// in there.
const IndexTemplate string = `{
	"template": "%s*",
	"order": 1,
	"mappings": {
		"_default_": {
			"_timestamp": {
				"enabled": true,
				"store": true
			},
			"dynamic_templates" : [{
				"label_name": {
					"path_match": "labels",
					"match_mapping_type": "string",
					"mapping": {
						"type": "string",
						"index": "not_analyzed"
					}
				}
			},
			{
				"company": {
					"match": "company",
					"match_mapping_type": "string",
					"mapping": {
						"type": "string",
						"index": "not_analyzed"
					}
				}
			},
			{
				"login": {
					"match": "login",
					"match_mapping_type": "string",
					"mapping": {
						"type": "string",
						"index": "not_analyzed"
					}
				}
			},
			{
				"milestone": {
					"match": "milestone",
					"match_mapping_type": "string",
					"mapping": {
						"type": "string",
						"index": "not_analyzed"
					}
				}
			},
			{
				"url": {
					"match": "*url",
					"match_mapping_type": "string",
					"mapping": {
						"type": "string",
						"index": "not_analyzed"
					}
				}
			}]
		}
	}
}`

func enableTimestamping(index string) error {
	return indices.PutMapping(index, "_default_", struct{}{}, indices.MappingOptions{
		Timestamp: indices.TimestampOptions{
			Enabled: true,
		},
	})
}

func doInitCommand(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))
	for _, r := range config.Repositories {
		if _, err := api.DoCommand("PUT", "/_template/events", nil, fmt.Sprintf(IndexTemplate, r.IndexPrefix())); err != nil {
			log.Fatal(err)
		}
	}
}
