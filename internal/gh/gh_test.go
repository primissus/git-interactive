package gh

import "testing"

func TestParsePRsOpenAndDraft(t *testing.T) {
	data := `[
		{"number":412,"state":"OPEN","isDraft":false,"title":"Add widget","headRefName":"feature/widget","url":"https://github.com/o/r/pull/412"},
		{"number":413,"state":"OPEN","isDraft":true,"title":"WIP thing","headRefName":"feature/wip","url":"https://github.com/o/r/pull/413"}
	]`
	prs, err := parsePRs([]byte(data))
	if err != nil {
		t.Fatalf("parsePRs: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("parsePRs = %d entries, want 2", len(prs))
	}
	if prs[0].Number != 412 || prs[0].IsDraft {
		t.Errorf("prs[0] = %+v, want number 412, not draft", prs[0])
	}
	if prs[1].Number != 413 || !prs[1].IsDraft {
		t.Errorf("prs[1] = %+v, want number 413, draft", prs[1])
	}
}

func TestParsePRsEmpty(t *testing.T) {
	prs, err := parsePRs([]byte(`[]`))
	if err != nil {
		t.Fatalf("parsePRs: %v", err)
	}
	if len(prs) != 0 {
		t.Errorf("parsePRs([]) = %v, want empty", prs)
	}
}

func TestParsePRsMalformed(t *testing.T) {
	if _, err := parsePRs([]byte(`{not json`)); err == nil {
		t.Error("parsePRs(malformed) = nil err, want error")
	}
}

func TestPRsByBranchKeysOnHeadRef(t *testing.T) {
	prs, err := parsePRs([]byte(`[{"number":1,"state":"OPEN","headRefName":"main-fix"}]`))
	if err != nil {
		t.Fatalf("parsePRs: %v", err)
	}
	out := make(map[string]PR, len(prs))
	for _, pr := range prs {
		out[pr.HeadRefName] = pr
	}
	if out["main-fix"].Number != 1 {
		t.Errorf("PRsByBranch keying = %+v, want number 1 under 'main-fix'", out)
	}
}
