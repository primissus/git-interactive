package git

import (
	"context"
	"testing"
)

func TestListTagsLightweightAndAnnotated(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")
	mustGit(t, r, "tag", "v1.0.0") // lightweight
	mustGit(t, r, "tag", "-a", "v2.0.0", "-m", "release v2")

	tags, err := ListTags(ctx, r)
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if len(tags) != 2 {
		t.Fatalf("want 2 tags, got %d: %+v", len(tags), tags)
	}

	byName := map[string]Tag{}
	for _, tg := range tags {
		byName[tg.Name] = tg
	}

	light, ok := byName["v1.0.0"]
	if !ok {
		t.Fatalf("missing v1.0.0: %+v", tags)
	}
	if light.Subject != "initial commit" {
		t.Errorf("lightweight tag subject = %q, want the commit subject", light.Subject)
	}
	if light.Author != "Test User" {
		t.Errorf("lightweight tag author = %q, want %q", light.Author, "Test User")
	}
	if !light.Head {
		t.Errorf("v1.0.0 should be marked Head (points at the checked-out commit)")
	}

	annotated, ok := byName["v2.0.0"]
	if !ok {
		t.Fatalf("missing v2.0.0: %+v", tags)
	}
	if annotated.Subject != "release v2" {
		t.Errorf("annotated tag subject = %q, want the tag message %q", annotated.Subject, "release v2")
	}
	if annotated.Author != "Test User" {
		t.Errorf("annotated tag author = %q, want tagger name %q", annotated.Author, "Test User")
	}
	if annotated.SHA != light.SHA {
		t.Errorf("annotated tag SHA = %q, want it dereferenced to the same commit as v1.0.0 (%q)", annotated.SHA, light.SHA)
	}
}

func TestTagExistsCreateDeleteCheckout(t *testing.T) {
	ctx := context.Background()
	r := newTestRepo(t)
	writeFile(t, r, "a.txt", "hello\n")
	mustGit(t, r, "add", ".")
	mustGit(t, r, "commit", "-m", "initial commit")

	if exists, err := TagExists(ctx, r, "v1"); err != nil || exists {
		t.Fatalf("TagExists(v1) before create = %v, %v; want false, nil", exists, err)
	}

	if err := CreateTag(ctx, r, "v1", ""); err != nil {
		t.Fatalf("CreateTag: %v", err)
	}
	if exists, err := TagExists(ctx, r, "v1"); err != nil || !exists {
		t.Fatalf("TagExists(v1) after create = %v, %v; want true, nil", exists, err)
	}

	if err := CheckoutTag(ctx, r, "v1"); err != nil {
		t.Fatalf("CheckoutTag: %v", err)
	}
	head, err := RevParse(ctx, r, "HEAD")
	if err != nil {
		t.Fatalf("RevParse(HEAD): %v", err)
	}
	tagSHA, err := RevParse(ctx, r, "v1")
	if err != nil || head != tagSHA {
		t.Fatalf("after checkout, HEAD = %q want %q (v1's sha), err=%v", head, tagSHA, err)
	}

	mustGit(t, r, "checkout", "main")
	if err := DeleteTag(ctx, r, "v1"); err != nil {
		t.Fatalf("DeleteTag: %v", err)
	}
	if exists, err := TagExists(ctx, r, "v1"); err != nil || exists {
		t.Fatalf("TagExists(v1) after delete = %v, %v; want false, nil", exists, err)
	}
}
