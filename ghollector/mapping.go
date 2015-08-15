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
	return map[string]interface{}{
		"template": pattern,
		"order":    1,
		"mappings": map[string]interface{}{
			"_default_": map[string]interface{}{
				"enabled": true,
				"store":   true,
			},
			"dynamic_templates": dynamicTemplates,
		},
	}
}

func notAnalyzedStringProto(pattern string) mappingProto {
	return map[string]interface{}{
		pattern: map[string]interface{}{
			"path_patch":         pattern,
			"match_mapping_type": "string",
			"mapping": map[string]interface{}{
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
		if _, err := api.DoCommand("put", "/_template/ghollector", nil, template); err != nil {
			log.Fatal(err)
		}
	}
}
