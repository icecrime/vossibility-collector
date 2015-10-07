package main

import (
	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/mattbaird/elastigo/api"
)

var syncMappingCommand = cli.Command{
	Name:   "sync_mapping",
	Usage:  "sync the configuration definition with the store mappings",
	Action: doSyncMapping,
}

type mappingProto map[string]interface{}

type templateProto struct {
	Template string `json:"template"`
	Order    int    `json:"order"`
	Mappings map[string]mappingProto
}

func makeTemplate(pattern string, dynamicTemplates []mappingProto) map[string]interface{} {
	return mappingProto{
		"template": pattern + "*",
		"order":    1,
		"mappings": mappingProto{
			"_default_": mappingProto{
				"_timestamp": mappingProto{
					"enabled": true,
					"store":   true,
				},
				"dynamic_templates": dynamicTemplates,
			},
		},
	}
}

func notAnalyzedStringProto(pattern string) mappingProto {
	return mappingProto{
		pattern: mappingProto{
			"match":              pattern,
			"match_mapping_type": "string",
			"mapping": mappingProto{
				"index": "not_analyzed",
				"type":  "string",
			},
		},
	}
}

// doSyncMapping synchronizes the configuration definition with the Elastic
// Search backend mappings.
func doSyncMapping(c *cli.Context) {
	config := ParseConfigOrDie(c.GlobalString("config"))

	notAnalyzedProtos := []mappingProto{}
	for _, notAnalyzedPattern := range config.NotAnalyzedPatterns {
		notAnalyzedProtos = append(notAnalyzedProtos, notAnalyzedStringProto(notAnalyzedPattern))
	}

	for _, r := range config.Repositories {
		template := makeTemplate(r.IndexPrefix(), notAnalyzedProtos)
		if _, err := api.DoCommand("PUT", "/_template/vossibility-"+r.GivenName, nil, template); err != nil {
			log.Fatal(err)
		}
	}
}
