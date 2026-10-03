package models

import (
	"reflect"
	"testing"
)

func TestDefaultFeedViews(t *testing.T) {
	want := []string{FeedViewDefault, FeedViewSimple, FeedViewCalendar}
	if got := DefaultFeedViews(); !reflect.DeepEqual(got, want) {
		t.Fatalf("DefaultFeedViews() = %#v, want %#v", got, want)
	}

	defaults := NewFeedDefaults()
	if !reflect.DeepEqual(defaults.Views, want) {
		t.Fatalf("NewFeedDefaults().Views = %#v, want %#v", defaults.Views, want)
	}
}

func TestFeedConfigApplyDefaultsInheritsViews(t *testing.T) {
	defaults := NewFeedDefaults()
	defaults.Views = []string{FeedViewDefault, FeedViewCalendar}

	feed := FeedConfig{}
	feed.ApplyDefaults(defaults)

	want := []string{FeedViewDefault, FeedViewCalendar}
	if !reflect.DeepEqual(feed.Views, want) {
		t.Fatalf("FeedConfig.Views = %#v, want %#v", feed.Views, want)
	}
	if !feed.HasView(FeedViewCalendar) {
		t.Fatal("expected inherited calendar view")
	}
	if feed.HasView(FeedViewSimple) {
		t.Fatal("did not expect simple view after site-level opt-out")
	}
}

func TestFeedConfigApplyDefaultsPreservesPerFeedViews(t *testing.T) {
	defaults := NewFeedDefaults()
	feed := FeedConfig{Views: []string{FeedViewDefault, FeedViewSimple}}

	feed.ApplyDefaults(defaults)

	want := []string{FeedViewDefault, FeedViewSimple}
	if !reflect.DeepEqual(feed.Views, want) {
		t.Fatalf("FeedConfig.Views = %#v, want %#v", feed.Views, want)
	}
	if feed.HasView(FeedViewCalendar) {
		t.Fatal("did not expect calendar view after per-feed opt-out")
	}
}

func TestHasViewUsesBuiltInDefaultsBeforeResolution(t *testing.T) {
	feed := FeedConfig{}
	for _, view := range DefaultFeedViews() {
		if !feed.HasView(view) {
			t.Fatalf("expected unresolved feed to expose %q", view)
		}
	}
}

func TestNewFeedConfigCopiesViewSlice(t *testing.T) {
	defaults := NewFeedDefaults()
	feed := NewFeedConfig(defaults)
	feed.Views[0] = "changed"
	if defaults.Views[0] != FeedViewDefault {
		t.Fatal("NewFeedConfig should copy defaults.Views instead of aliasing it")
	}
}
