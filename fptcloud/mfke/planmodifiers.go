package fptcloud_mfke

import (
	"context"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// PlanModifier để reorder pools theo name
type reorderByNameModifier struct{}

func (m reorderByNameModifier) Description(ctx context.Context) string {
	return "Reorder pools by name to avoid index shift"
}

func (m reorderByNameModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m reorderByNameModifier) PlanModifyList(ctx context.Context, req planmodifier.ListRequest, resp *planmodifier.ListResponse) {
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		return
	}

	var planElems []types.Object
	diags := req.PlanValue.ElementsAs(ctx, &planElems, false)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Sort theo worker_base trước, rồi đến name
	sort.SliceStable(planElems, func(i, j int) bool {
		// Lấy worker_base
		workerBaseI := planElems[i].Attributes()["worker_base"].(types.Bool).ValueBool()
		workerBaseJ := planElems[j].Attributes()["worker_base"].(types.Bool).ValueBool()

		// Sắp xếp theo worker_base trước (true trước)
		if workerBaseI != workerBaseJ {
			return workerBaseI
		}

		// Nếu worker_base giống nhau, sắp xếp theo name
		nameI := planElems[i].Attributes()["name"].(types.String).ValueString()
		nameJ := planElems[j].Attributes()["name"].(types.String).ValueString()
		return nameI < nameJ
	})

	newVal, diags := types.ListValueFrom(ctx, req.PlanValue.ElementType(ctx), planElems)
	resp.Diagnostics.Append(diags...)
	if !resp.Diagnostics.HasError() {
		resp.PlanValue = newVal
	}
}

func listReorderByName() planmodifier.List {
	return reorderByNameModifier{}
}
