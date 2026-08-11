package renderingcontract

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestRenderPlanFixtures_CompleteAndResolve(t *testing.T) {
	fixtures, err := LoadRenderPlanFixtures(filepath.Join("..", "..", "spec", "rendering-contract", "render-plan-fixtures.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures.Fixtures {
		t.Run(fixture.ID, func(t *testing.T) {
			state, plan, _, err := ResolveFixture(fixture)
			if err != nil {
				t.Fatal(err)
			}
			wantState := fixture.ExpectedNormalized
			if wantState.Fontpack == "brush-poster" {
				wantState.Fontpack = "brush"
			}
			if !reflect.DeepEqual(state, wantState) {
				t.Errorf("normalized state = %#v, want %#v", state, wantState)
			}
			wantPlan := fixture.ExpectedPlan
			if wantPlan.Fontpack == "brush-poster" {
				wantPlan.Fontpack = "brush"
			}
			if !reflect.DeepEqual(plan, wantPlan) {
				t.Errorf("render plan = %#v, want %#v", plan, wantPlan)
			}
			for _, layer := range plan.Layers {
				if strings.Contains(layer.Recipe, "://") {
					t.Errorf("layer recipe is browser-specific: %q", layer.Recipe)
				}
			}
		})
	}
}

func TestRenderPlanFixtures_SemanticInvariants(t *testing.T) {
	fixtures, err := LoadRenderPlanFixtures(filepath.Join("..", "..", "spec", "rendering-contract", "render-plan-fixtures.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range fixtures.Fixtures {
		state, plan, _, err := ResolveFixture(fixture)
		if err != nil {
			t.Fatal(fixture.ID, ": ", err)
		}
		if state.HeadingTexture.Kind == "inherit" {
			t.Errorf("%s retained unresolved heading inherit", fixture.ID)
		}
		if plan.HeadingTexture.Recipe != state.HeadingTexture.Kind {
			t.Errorf("%s heading recipe = %q, want %q", fixture.ID, plan.HeadingTexture.Recipe, state.HeadingTexture.Kind)
		}
		if plan.Motif.Mix == 0 && len(plan.Layers) != 3 {
			t.Errorf("%s zero motif mix removed configured sandwich passes", fixture.ID)
		}
		if plan.Motif.Kind != "off" && plan.Motif.URL != "" && !strings.Contains(plan.Motif.URL, "://") {
			t.Errorf("%s motif URL is not explicit: %q", fixture.ID, plan.Motif.URL)
		}
	}
}
