# Readme

This directory contains the example from the accompanying presentation.

https://play.openpolicyagent.org/p/VlDtOzzxX8

# Show the org chart

```sh
$ opa eval data.permissions.org_chart_graph -d permissions.rego -f raw | jq
{
  "boss": [
    "developers",
    "human_resources"
  ],
  "developers": [
    "interns"
  ],
  "human_resources": [],
  "interns": []
}
```

## Show the permissions for the org chart

```sh
$ opa eval data.permissions.org_chart_permissions -d permissions.rego -f raw | jq
{
  "boss": [
    "dev",
    "prod",
    "salaries"
  ],
  "developers": [
    "dev",
    "prod"
  ],
  "human_resources": [
    "salaries"
  ],
  "interns": [
    "dev"
  ]
}
```

## Show the result of allow

```sh
$ cat input.json | jq
{
    "business_unit": "developers",
    "request": "salaries"
}

$ opa eval data.permissions.allow -d permissions.rego -f raw -i input.json | jq
false
```