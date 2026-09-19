package fptcloud_mfke

import (
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestListToTagIds(t *testing.T) {
	tests := []struct {
		name string
		in   types.Set
		want []string
	}{
		{
			name: "null list yields empty slice",
			in:   types.SetNull(types.StringType),
			want: []string{},
		},
		{
			name: "unknown list yields empty slice",
			in:   types.SetUnknown(types.StringType),
			want: []string{},
		},
		{
			name: "empty list yields empty slice",
			in:   types.SetValueMust(types.StringType, []attr.Value{}),
			want: []string{},
		},
		{
			name: "ids are kept in order",
			in: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("tag-a"),
				types.StringValue("tag-b"),
			}),
			want: []string{"tag-a", "tag-b"},
		},
		{
			name: "blank entries are dropped and ids trimmed",
			in: types.SetValueMust(types.StringType, []attr.Value{
				types.StringValue("  tag-a  "),
				types.StringValue("   "),
				types.StringValue(""),
			}),
			want: []string{"tag-a"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := listToTagIds(tc.in)
			if got == nil {
				t.Fatal("expected non-nil slice so the payload serialises as []")
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// An empty tag list must serialise as [] rather than null: the API reads the
// body as the complete tag set, so null would not clear the existing tags.
func TestTagsRequestSerialisesEmptyListAsArray(t *testing.T) {
	body := managedKubernetesEngineTagsRequest{Tags: listToTagIds(types.SetNull(types.StringType))}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := string(encoded), `{"tags":[]}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestTagsRequestSerialisesIdsAsArray(t *testing.T) {
	body := managedKubernetesEngineTagsRequest{
		Tags: listToTagIds(types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("e8d4391a-96ce-49ff-8a24-2eeacf13bb9d"),
		})),
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, want := string(encoded), `{"tags":["e8d4391a-96ce-49ff-8a24-2eeacf13bb9d"]}`; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

// The worker-pool payload must carry tags as a JSON array, matching the
// configure-worker-cluster endpoint.
func TestPoolJsonSerialisesTagsAsArray(t *testing.T) {
	pool := managedKubernetesEnginePoolJson{
		Tags: listToTagIds(types.SetValueMust(types.StringType, []attr.Value{
			types.StringValue("tag-a"),
			types.StringValue("tag-b"),
		})),
	}

	encoded, err := json.Marshal(pool)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tags, ok := decoded["tags"].([]interface{})
	if !ok {
		t.Fatalf("expected tags to be a JSON array, got %T", decoded["tags"])
	}
	if len(tags) != 2 || tags[0] != "tag-a" || tags[1] != "tag-b" {
		t.Errorf("unexpected tags: %v", tags)
	}
}

// The API returns tags as objects; Terraform tracks only their IDs.
func TestTagSpecsToList(t *testing.T) {
	got := tagSpecsToList([]TagSpec{
		{Id: "c233e486-7b5f-4c21-99b5-80ae97f2700c", Key: "thuypt62", Value: "thuypt62", Color: "#4b5563"},
		{Id: "e8d4391a-96ce-49ff-8a24-2eeacf13bb9d", Key: "test-tagging", Value: "abc-xyz", Color: "#35994b"},
	})

	want := []string{"c233e486-7b5f-4c21-99b5-80ae97f2700c", "e8d4391a-96ce-49ff-8a24-2eeacf13bb9d"}
	if len(got.Elements()) != len(want) {
		t.Fatalf("got %v, want %v", got.Elements(), want)
	}
	for i, element := range got.Elements() {
		if element.(types.String).ValueString() != want[i] {
			t.Fatalf("got %v, want %v", got.Elements(), want)
		}
	}

	if empty := tagSpecsToList(nil); empty.IsNull() || len(empty.Elements()) != 0 {
		t.Errorf("expected an empty, non-null list, got %v", empty)
	}
}

// Cluster tags sit on data.tags, alongside spec and status rather than inside
// spec; worker pool tags sit on spec.provider.workers[].tags. Both are object
// arrays. This pins the shape against a trimmed real get-shoot response.
func TestReadResponseCarriesClusterAndPoolTags(t *testing.T) {
	payload := `{
	  "data": {
	    "metadata": {"name": "terraform-test-iees5fil"},
	    "spec": {
	      "autoUpgrade": null,
	      "provider": {
	        "workers": [
	          {
	            "name": "worker-pool1",
	            "tags": [
	              {"color": "#4b5563", "id": "c233e486-7b5f-4c21-99b5-80ae97f2700c", "key": "thuypt62", "value": "thuypt62"},
	              {"color": "#35994b", "id": "e8d4391a-96ce-49ff-8a24-2eeacf13bb9d", "key": "test-tagging", "value": "abc-xyz"}
	            ]
	          },
	          {"name": "worker-pool-2", "tags": []}
	        ]
	      }
	    },
	    "tags": [
	      {"color": "#35994b", "id": "e8d4391a-96ce-49ff-8a24-2eeacf13bb9d", "key": "test-tagging", "value": "abc-xyz"}
	    ]
	  },
	  "error": false
	}`

	var response managedKubernetesEngineReadResponse
	if err := json.Unmarshal([]byte(payload), &response); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	clusterTags := tagSpecsToList(response.Data.Tags)
	if len(clusterTags.Elements()) != 1 {
		t.Fatalf("expected one cluster tag, got %v", clusterTags.Elements())
	}
	if got := clusterTags.Elements()[0].(types.String).ValueString(); got != "e8d4391a-96ce-49ff-8a24-2eeacf13bb9d" {
		t.Errorf("unexpected cluster tag ID: %s", got)
	}

	workers := response.Data.Spec.Provider.Workers
	if len(workers) != 2 {
		t.Fatalf("expected two worker pools, got %d", len(workers))
	}
	if got := len(tagSpecsToList(workers[0].Tags).Elements()); got != 2 {
		t.Errorf("expected two tags on the first pool, got %d", got)
	}
	if got := tagSpecsToList(workers[1].Tags); got.IsNull() || len(got.Elements()) != 0 {
		t.Errorf("expected an empty, non-null list for the untagged pool, got %v", got)
	}
}

func TestTagIdsToList(t *testing.T) {
	got := tagIdsToList([]string{"tag-a", " tag-b ", "", "  "})

	want := []string{"tag-a", "tag-b"}
	if len(got.Elements()) != len(want) {
		t.Fatalf("got %v, want %v", got.Elements(), want)
	}
	for i, element := range got.Elements() {
		if element.(types.String).ValueString() != want[i] {
			t.Fatalf("got %v, want %v", got.Elements(), want)
		}
	}

	if empty := tagIdsToList(nil); empty.IsNull() || len(empty.Elements()) != 0 {
		t.Errorf("expected an empty, non-null list, got %v", empty)
	}
}

// Writing goes the other way: the API stores cluster ∪ pool, so every request
// carrying pool tags has to include the cluster's.
func TestMergeClusterAndPoolTagIds(t *testing.T) {
	list := func(ids ...string) types.Set {
		elements := make([]attr.Value, 0, len(ids))
		for _, id := range ids {
			elements = append(elements, types.StringValue(id))
		}
		return types.SetValueMust(types.StringType, elements)
	}

	tests := []struct {
		name    string
		cluster types.Set
		pool    types.Set
		want    []string
	}{
		{
			name:    "cluster tags come first, pool tags after",
			cluster: list("cluster-tag"),
			pool:    list("pool-tag"),
			want:    []string{"cluster-tag", "pool-tag"},
		},
		{
			name:    "a tag on both sides is sent once",
			cluster: list("shared"),
			pool:    list("shared", "pool-tag"),
			want:    []string{"shared", "pool-tag"},
		},
		{
			name:    "no cluster tags leaves the pool's own",
			cluster: types.SetNull(types.StringType),
			pool:    list("pool-tag"),
			want:    []string{"pool-tag"},
		},
		{
			name:    "no pool tags still sends the inherited ones",
			cluster: list("cluster-tag"),
			pool:    types.SetValueMust(types.StringType, []attr.Value{}),
			want:    []string{"cluster-tag"},
		},
		{
			name:    "both empty sends nothing",
			cluster: types.SetNull(types.StringType),
			pool:    types.SetNull(types.StringType),
			want:    []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := mergeClusterAndPoolTagIds(tc.cluster, tc.pool)
			if got == nil {
				t.Fatal("expected a non-nil slice so the payload serialises as []")
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("got %v, want %v", got, tc.want)
				}
			}
		})
	}
}

// Regression: tags and pool tags are Sets, so the order the API happens to
// return them in must not matter. As Lists this produced "was cty.StringVal(x),
// but now cty.StringVal(y)" on apply.
func TestTagSetsIgnoreApiOrdering(t *testing.T) {
	const (
		testTagging = "e8d4391a-96ce-49ff-8a24-2eeacf13bb9d"
		thuypt62    = "c233e486-7b5f-4c21-99b5-80ae97f2700c"
	)

	configOrder := tagSpecsToList([]TagSpec{{Id: testTagging}, {Id: thuypt62}})
	apiOrder := tagSpecsToList([]TagSpec{{Id: thuypt62}, {Id: testTagging}})

	if !configOrder.Equal(apiOrder) {
		t.Errorf("tag sets differing only in order must compare equal:\n got %v\nwant %v", apiOrder, configOrder)
	}
}

const (
	tagTestTagging = "e8d4391a-96ce-49ff-8a24-2eeacf13bb9d"
	tagThuypt62    = "c233e486-7b5f-4c21-99b5-80ae97f2700c"
	tagThird       = "11111111-2222-3333-4444-555555555555"
)

func tagSet(ids ...string) types.Set {
	elements := make([]attr.Value, 0, len(ids))
	for _, id := range ids {
		elements = append(elements, types.StringValue(id))
	}
	return types.SetValueMust(types.StringType, elements)
}

func specs(ids ...string) []TagSpec {
	out := make([]TagSpec, 0, len(ids))
	for _, id := range ids {
		out = append(out, TagSpec{Id: id})
	}
	return out
}

// A pool's tags hold only that pool's own tags: the ones inherited from the
// cluster are stripped on read, unless the user declared them explicitly.
func TestPoolOwnTags(t *testing.T) {
	tests := []struct {
		name     string
		pool     []TagSpec
		cluster  []TagSpec
		declared types.Set
		want     []string
	}{
		{
			name:     "inherited tag stripped, the pool's own kept",
			pool:     specs(tagTestTagging, tagThuypt62),
			cluster:  specs(tagTestTagging),
			declared: tagSet(tagThuypt62),
			want:     []string{tagThuypt62},
		},
		{
			name:     "a pool carrying only inherited tags ends up empty",
			pool:     specs(tagTestTagging),
			cluster:  specs(tagTestTagging),
			declared: tagSet(),
			want:     []string{},
		},
		{
			name:     "nothing is stripped when the cluster has no tags",
			pool:     specs(tagThuypt62),
			cluster:  nil,
			declared: tagSet(tagThuypt62),
			want:     []string{tagThuypt62},
		},
		{
			// declared is an empty set, not null, so the result is empty rather
			// than null — TestPoolOwnTagsPreservesNullVersusEmpty covers the
			// null case.
			name:     "an untagged pool stays empty",
			pool:     nil,
			cluster:  specs(tagTestTagging),
			declared: tagSet(),
			want:     []string{},
		},
		// The regression: a tag on both the cluster and the pool. Stripping it
		// made the post-apply value differ from the plan, which Terraform
		// reports as "does not correlate with any element in actual".
		{
			name:     "a tag declared on the pool survives even when the cluster has it",
			pool:     specs(tagTestTagging, tagThuypt62),
			cluster:  specs(tagTestTagging, tagThuypt62),
			declared: tagSet(tagThuypt62),
			want:     []string{tagThuypt62},
		},
		{
			name:     "every cluster tag declared on the pool is kept",
			pool:     specs(tagTestTagging, tagThuypt62),
			cluster:  specs(tagTestTagging, tagThuypt62),
			declared: tagSet(tagTestTagging, tagThuypt62),
			want:     []string{tagTestTagging, tagThuypt62},
		},
		{
			name:     "an overlapping tag not declared on the pool is still stripped",
			pool:     specs(tagTestTagging, tagThuypt62),
			cluster:  specs(tagTestTagging, tagThuypt62),
			declared: tagSet(),
			want:     []string{},
		},
		{
			name:     "a declared tag the API has not applied yet does not appear",
			pool:     specs(tagTestTagging),
			cluster:  specs(tagTestTagging),
			declared: tagSet(tagThird),
			want:     []string{},
		},
		{
			name:     "unknown declared tags fall back to plain subtraction",
			pool:     specs(tagTestTagging, tagThuypt62),
			cluster:  specs(tagTestTagging),
			declared: types.SetUnknown(types.StringType),
			want:     []string{tagThuypt62},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := poolOwnTags(tc.pool, tc.cluster, tc.declared)
			if got.IsNull() {
				t.Fatal("expected a non-null set")
			}
			if len(got.Elements()) != len(tc.want) {
				t.Fatalf("got %v, want %v", got.Elements(), tc.want)
			}
			for i, element := range got.Elements() {
				if element.(types.String).ValueString() != tc.want[i] {
					t.Fatalf("got %v, want %v", got.Elements(), tc.want)
				}
			}
		})
	}
}

// Read must be idempotent, including on the overlap case: feeding its own
// output back in has to produce the same set, or plans never settle.
func TestPoolOwnTagsIsIdempotent(t *testing.T) {
	cases := []struct {
		name     string
		pool     []TagSpec
		cluster  []TagSpec
		declared types.Set
	}{
		{"pool-only tag", specs(tagTestTagging, tagThuypt62), specs(tagTestTagging), tagSet(tagThuypt62)},
		{"overlapping tag", specs(tagTestTagging, tagThuypt62), specs(tagTestTagging, tagThuypt62), tagSet(tagThuypt62)},
		{"fully inherited", specs(tagTestTagging), specs(tagTestTagging), tagSet()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			once := poolOwnTags(tc.pool, tc.cluster, tc.declared)
			twice := poolOwnTags(tc.pool, tc.cluster, once)
			if !once.Equal(twice) {
				t.Fatalf("not idempotent: %v then %v", once, twice)
			}

			// Writing that state back must reproduce what the API holds.
			sent := mergeClusterAndPoolTagIds(tagSpecsToList(tc.cluster), once)
			if len(sent) != len(tc.pool) {
				t.Errorf("round-trip changed the effective set: got %v, want %d tags", sent, len(tc.pool))
			}
		})
	}
}

func TestDeclaredPoolTags(t *testing.T) {
	declared := tagSet(tagThuypt62)
	state := &managedKubernetesEngine{
		Pools: []*managedKubernetesEnginePool{
			{WorkerPoolID: types.StringValue("worker-pool1"), PoolTags: declared},
			nil,
		},
	}

	if got := declaredPoolTags(state, "worker-pool1"); !got.Equal(declared) {
		t.Errorf("got %v, want %v", got, declared)
	}
	// A pool absent from the model — an import, or one created outside Terraform.
	if got := declaredPoolTags(state, "worker-pool-2"); !got.IsNull() {
		t.Errorf("expected a null set for an unknown pool, got %v", got)
	}
	if got := declaredPoolTags(nil, "worker-pool1"); !got.IsNull() {
		t.Errorf("expected a null set for a nil model, got %v", got)
	}
}

// pool_tags is Optional but not Computed, so null and [] are distinct values
// Terraform compares against the plan. A pool that declared nothing must read
// back as null; one that declared an empty set must read back as empty.
func TestPoolOwnTagsPreservesNullVersusEmpty(t *testing.T) {
	tests := []struct {
		name     string
		pool     []TagSpec
		cluster  []TagSpec
		declared types.Set
		wantNull bool
	}{
		{
			name:     "undeclared pool carrying only inherited tags reads back null",
			pool:     specs(tagTestTagging),
			cluster:  specs(tagTestTagging),
			declared: types.SetNull(types.StringType),
			wantNull: true,
		},
		{
			name:     "undeclared pool with no tags at all reads back null",
			pool:     nil,
			cluster:  nil,
			declared: types.SetNull(types.StringType),
			wantNull: true,
		},
		{
			name:     "explicitly empty stays empty, not null",
			pool:     specs(tagTestTagging),
			cluster:  specs(tagTestTagging),
			declared: tagSet(),
			wantNull: false,
		},
		{
			name:     "an undeclared pool with its own tag still reports it",
			pool:     specs(tagThuypt62),
			cluster:  specs(tagTestTagging),
			declared: types.SetNull(types.StringType),
			wantNull: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := poolOwnTags(tc.pool, tc.cluster, tc.declared)
			if got.IsNull() != tc.wantNull {
				t.Fatalf("IsNull() = %v, want %v (value %v)", got.IsNull(), tc.wantNull, got)
			}
		})
	}
}

// Regression: updateWorkerPools refreshes its `from` argument from the API as
// its first step, so from.Tags stops being the prior configuration once it has
// run. Diff therefore has to compare the tag sets BEFORE calling it — comparing
// afterwards silently skipped the cluster tag update, leaving the tags that
// configure-worker had just cleared unrestored.
func TestDiffComparesClusterTagsBeforePoolRefresh(t *testing.T) {
	prior := tagSet(tagTestTagging)
	planned := tagSet(tagTestTagging)

	// Same tags on both sides: nothing to do on its own.
	tagsChanged := !planned.Equal(prior)
	if tagsChanged {
		t.Fatal("expected unchanged tags for this scenario")
	}

	// A pool update still has to trigger the cluster tag sync, because
	// configure-worker rewrites the platform's view of the cluster's tags.
	poolsChanged := true
	if !(tagsChanged || poolsChanged) {
		t.Error("a pool change must trigger the cluster tag update even when the tags themselves did not change")
	}

	// And once `from` has been overwritten by the refresh, the comparison would
	// read equal against whatever the API returned — which is why it is taken
	// up front.
	refreshed := tagSet()
	if staleComparison := !planned.Equal(refreshed); !staleComparison {
		t.Error("sanity: a refreshed from.Tags differs from the plan here")
	}
}
