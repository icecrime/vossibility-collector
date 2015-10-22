vossibility-collector
---------------------

Vossibility-collector is the core component of vossibility, a project providing better visibility
for your open source project. The project was initially started for [Docker](https://docker.io) but
is not tied to it in any way.

# Overview

Vossibility-collector receives live GitHub data from a [NSQ](http://nsq.io/) queue and the [GitHub
API](https://developer.github.com/v3/) on one end, and feeds structured data to Elastic Search on
the other end. It provides:

- The power of Elastic Search to search into your repository (e.g., "give me all pull requests
comments from a user who's name is approximately ice and contains LGTM").  - The power of Kibana to
build dashboards for your project: a basic example of what can be achieved is shown below.

![Sample dashboard](https://github.com/icecrime/vossibility-collector/raw/master/resources/dashboard.png)

# Usage

Also see [Environment setup](#environment-setup)

```
NAME:
   vossibility-collector - collect GitHub repository data

USAGE:
   vossibility-collector [global options] command [command options] [arguments...]
   
VERSION:
   0.1.0
   
COMMANDS:
   limits       get information about your GitHub API rate limits
   run          listen and process GitHub events
   sync         sync storage with the GitHub repositories
   sync_mapping sync the configuration definition with the store mappings
   sync_users   sync the user store with the information from a file
   help, h      Shows a list of commands or help for one command
   
GLOBAL OPTIONS:
   -c, --config "config.toml"   configuration file
   --debug                      enable debug output
   --debug-es                   enable debug output for elasticsearch queries
   --help, -h                   show help
   --version, -v                print the version
```

# Configuration

Because different project may want to track different data, and that volumetry on [high activity
projects](https://github.com/docker/docker) makes it unreasonable to store every bits of
information, vossibility-collector was created to be highly configurable.

The project use the `toml` file format for its configurations. Example files can be found in the
[`examples/`](https://github.com/icecrime/vossibility-collector/tree/master/examples) directory.

### Top-level keys

Element            | Type   | Description
------------------ | -------|------------
`elasticsearch`    | String | Host or address for the Elastic Search server
`github_api_token` | String | Optional GitHub API token (important for [API rate limiting](https://developer.github.com/v3/#rate-limiting))
`sync_periodicity` | String | Interval at which the complete state of repositories are synced (`hourly`, `daily`, or `weekly`)

### `[nsq]` section

The `[nsq]` section defines configuration relative to the NSQ queue.

Element            | Type   | Description
------------------ | -------|------------
`channel`          | String | NSQ channel to listen on
`lookupd`          | String | Address of the lookupd server

### `[mapping]` section

The `[mapping]` section defines configuration relative to the Elastic Search [mapping
mechanism](https://www.elastic.co/guide/en/elasticsearch/guide/current/mapping-intro.html). Right
now, it supports a single value of type string array that defines a list of patterns that should
_not_ be analyzed by Elastic Search. In particular, anything that relates to labels, user names, or
company names, are things that you most likely do not want analyzed using the default analyzer.

Element            | Type   | Description
------------------ | -------|------------
`not_analyzed`     | String array | Patterns to exclude from analyzer

### `[repositories]` section

The `[repositories]` section defines a collection of tables (in [toml
terminology](https://github.com/toml-lang/toml#table)), where [each of them](#repository-definition)
defines a GitHub repository for which data should be collected.

#### Repository definition

Element            | Type   | Description
------------------ | -------|------------
`user`             | String | GitHub username
`repo`             | String | GitHub repository
`topic`            | String | NSQ topic to listen on for this repository
`events`           | String | Optional reference to an [event set](#[event_set]-section) (defaults to `"default"`)
`start_index`      | String | Optional starting issues # for high activity repositories

### `[event_set]` section

The `[event_set]` section defines a collection of tables (in [toml
terminology](https://github.com/toml-lang/toml#table)), where [each of them](#event-set-definition)
defines a list of GitHub events to subscribe to, along with its related
[transformation](#transformation-definition).

#### Event set definition

In an event set definition, each key should be a valid [GitHub event
identifier](https://developer.github.com/webhooks/#events), and each value a string that references
a given [transformation](#transformation-definition).

We also require two special entries in an event set definition: `snapshot_issue` (which applies to
snapshoted issue content) and `snapshot_pull_request` (which applies to snapshoted pull request
content). Those are mandatory because we assume that issues and pull requests are always being
stored.

### `[transformations]` section

The `[transformation]` section is both the most complex and most interesting section. It defines a
collection of tables (in [toml terminology](https://github.com/toml-lang/toml#table)), where [each
of them](#transformation-definition) defines the data model for storing a particular GitHub object.

#### Transformation definition

A GitHub event is very rich. For example, a simple [pull request
event](https://developer.github.com/v3/activity/events/types/#pullrequestevent) has hundreds of
fields, most of them you'll probably never query or use. That's the point of a transformation: be
able to define your very own data model for the things you care about, and provide a
configuration-style definition of the way that data should be filled.

For this, the project relies on a [modified
version](https://github.com/icecrime/vossibility-collector/tree/master/src/github.com/icecrime/template)
of the standard [`text/template`](http://golang.org/pkg/text/template/) Go package. This allows you
to use the standard Go template DSL to describe your data mapping (think of it as a text-based API
for runtime reflection).

#### Transformation example

Let's take a concrete example:

```
    [transformations.issue]
    assignee = "{{ .assignee }}"
    author = "{{ user_data .user.login }}"
    body = "{{ .body }}"
    closed_at = "{{ .closed_at }}"
    comments = "{{ .comments }}"
    created_at = "{{ .created_at }}"
    labels = "{{ range .labels }}{{ .name }}{{ end }}"
    locked = "{{ .locked }}"
    milestone = "{{ if .milestone }}{{ .milestone.title }}{{end}}"
    number = "{{ .number }}"
    state = "{{ .state }}"
    updated_at = "{{ .updated_at }}"
```

In a transformation definition, the resulting data structure will have one field per key in the
table. In this case, the result of applying that transformation to a GitHub object will be a
structure with an `assignee` field, which values is the result of applying the template `{{ .
assignee }}` to the source object. This is a trivial field: the output value is directly that of the
input.

But templates can do much more! One thing they can do is flow control, such as loops or tests. For
example, the `milestone` field in a GitHub issue can either be `nil`, or an object containing
multiple fields, or which we only care about the `title` in that particular example. The template
language allow us to express that we specifically care about the `title` field for a non-`nil`
milestone, in all other cases the resulting `milestone` field will be `nil`.

Same goes for labels, which is an array of objects in the GitHub model, but in most cases we really
just care about the label `name` rather than its `url` or `color` code. The standard template
`range` construct allow us to loop on each label and extract specifically the field we're interested
in. In that case, the output `labels` field will be an array of strings, not an array of objects.

Finally, the template language supports functions. Looking at the `author` field, you'll see that
its value should be the result of evaluating `{{ user_data .user.login }}`. That means: apply the
function `user_data` to the result of evaluating `.user.login` on the source object (`user_data` is
a [builtin function](#builtin-functions) described below).

#### Builtin functions

We currently support the following builtin functions:

- `apply_transformation` takes a single argument and uses this as the name of a transformation to
apply to a sub element of the source object. This is for example particularly useful for a GitHub
[`issue_event`](https://developer.github.com/v3/activity/events/types/#issuesevent) that contains a
nested issue object on which we'd like to apply the same transformation we usually do.

- `context` is a constant function that returns contextual information about the data being
processed. It currently has a single field `Repository` that exposes two methods: `FullName` (e.g.,
`docker/docker`) and `PrettyName` (e.g., `engine (docker:docker)`).

- `days_difference` computes the difference between two GitHub formatted dates and returns the
result as a floating number of days.

- `user_data` takes a single argument, uses its value as a document identifier in the to query an
object of type `user` in the index `users` of the Elastic Search backend, and returns the source
JSON.
  
  This effectively allows to enrich the value of a field with information in database: in our case,
this allows to replace a single login such as `icecrime` into a structure object that contains
information about the person's employer and maintainer status.

# Project state

The project is in a very early state, and most notably lacks testing and CI.


# Environment setup

#### Pull docker images
```
docker pull nsqio/nsq
docker pull elasticsearch
docker pull pblittle/docker-logstash
docker pull icecrime/vossibility-collector
```

#### Create the vossibility config file
Create a [config file](https://github.com/icecrime/vossibility-collector/blob/master/examples/config.toml.example) as noted above.  You can create a local copy in your home directory, and reference it on launch (details below).

#### Create a data mount for NSQ
`docker create -v /data --name data nsqio/nsq /bin/true`

#### Launch NSQ
```
docker run -d --name lookupd -p 4160:4160 -p 4161:4161 nsqio/nsq /nsqlookupd
docker run -d --name nsqd -p 4150:4150 -p 4151:4151 nsqio/nsq /nsqd --data-path=/data --broadcast-address=172.17.42.1 --lookupd-tcp-address=172.17.42.1:4160
```

Use `docker inspect lookupd | grep Gateway` to get the IP address, then...

Test with `curl -X GET http://{nsq-gateway-address}:4151/info`.  You should see something like:

```
{"status_code":200,"status_txt":"OK","data":{"version":"0.3.6","broadcast_address":"172.17.42.1","hostname":"a0c7754cd9bc","http_port":4151,"tcp_port":4150,"start_time":1445533155}}
```

#### Create NSQ topics
There should be a topic for each repository listed in `config.toml`.  After launching NSQ, use the following command to create a topic:

`curl -X POST http://{nsq-gateway-address}:4151/topic/create?topic={name-of-topic}`

where `{name-of-topic}` should match the repository topic specified in `config.toml`

Test with `curl -X GET http://{nsq-gateway-address}:4151/stats`.  You should see something like:

```
nsqd v0.3.6 (built w/go1.5.1)
start_time 2015-10-22T16:59:15Z
uptime 2m52.615799031s

Health: OK

   [{name-of-topic}] depth: 0     be-depth: 0     msgs: 0        e2e%:

```

#### Launch ElasticSearch
`docker run -d --name elasticsearch elasticsearch`

Use `docker inspect elasticsearch | grep IPAddress` to get the IP address, then...

Test with `curl -X GET http://{elasticsearch-ip-address}:9200`.  You should see something like:

```
{
  "status" : 200,
  "name" : "Cable",
  "cluster_name" : "elasticsearch",
  "version" : {
    "number" : "1.7.3",
    "build_hash" : "05d4530971ef0ea46d0f4fa6ee64dbc8df659682",
    "build_timestamp" : "2015-10-15T09:14:17Z",
    "build_snapshot" : false,
    "lucene_version" : "4.10.4"
  },
  "tagline" : "You Know, for Search"
}
```

#### Set up Logstash
`docker run -d --name logstash -e ES_PORT=9200 -p 9200:9200 -p 9292:9292 pblittle/docker-logstash`

Use `ifconfig | grep addr` to find the IP address bound to the host adapter (typically this is the address bound to 127.0.0.1), then...

Test with:
`curl -X GET http://{host-adapter-nat-address}:9292/index.html#/dashboard/file/default.json`
You should a block of HTML returned.

#### Set up vossibility

##### Check host network settings
VirtualBox on MS Windows:
![vossibility network setup](https://raw.githubusercontent.com/JacquesPerrault/jacquesperrault.github.io/master/images/vossibility-network-settings.jpg)

##### Run vossibility
`docker run -v {local-config-file}:/etc/config.toml -p 4140:4140 -p 4141:4141 --name vossibility icecrime/vossibility-collector -c "/etc/config.toml" run`

Note that `{local-config-file}` might be something like `/home/docker/config.toml`

#### Launch Kibana
Give Logstash a minute or so to start up, then open a browser on the host machine to
`http://{host-adapter-nat-address}:9292/index.html#/dashboard/file/default.json`

Hint: this is the the same as the logstash curl test.  You should see something like:
![Kibana welcome screen](https://raw.githubusercontent.com/JacquesPerrault/jacquesperrault.github.io/master/images/kibana-welcome-screen.jpg)