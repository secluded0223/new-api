package authz

const ActionMerge = "merge"

const ResourceToken = "token"

var (
	TokenRead  = Permission{Resource: ResourceToken, Action: ActionRead}
	TokenMerge = Permission{Resource: ResourceToken, Action: ActionMerge}
)

func init() {
	RegisterResource(ResourceDefinition{
		Resource: ResourceToken,
		LabelKey: "Employee API key management",
		Actions: []ActionDefinition{
			{
				Action:         ActionRead,
				LabelKey:       "View employee API keys",
				DescriptionKey: "View masked API keys and usage for users below your role.",
			},
			{
				Action:         ActionMerge,
				LabelKey:       "Merge API key consumption",
				DescriptionKey: "Move one key's cumulative consumption to another key without changing quota or remaining balance.",
			},
		},
	})
}
