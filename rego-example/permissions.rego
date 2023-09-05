package permissions

import future.keywords.in

things = 1

org_chart := {
	"boss": {},
	"human_resources": {
		"managed_by": "boss",
		"access": ["salaries"],
	},
	"developers": {
		"managed_by": "boss",
		"access": ["prod", "dev"],
	},
	"interns": {
		"managed_by": "developers",
		"access": ["dev"],
	},
}

org_chart_graph[business_unit] := edges {
	some business_unit, _ in org_chart
	edges := {neighbour_ref |
		some neighbour_ref, neighbour in org_chart
		neighbour.managed_by == business_unit
	}
}

org_chart_permissions[business_unit] := access {
	some business_unit, _ in org_chart
	reachable := graph.reachable(org_chart_graph, {business_unit})

	access := {item |
		some bu, _ in reachable
		some resource, item in org_chart[bu].access
	}
}

default allow := false

allow {
	input.request in org_chart_permissions[input.business_unit]
}
