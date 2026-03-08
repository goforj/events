package main

import "testing"

func TestUpdateTestsSection(t *testing.T) {
	body := `before
<!-- test-count:embed:start -->
old
<!-- test-count:embed:end -->
after`

	got, err := updateTestsSection(body, Counts{Unit: 12, Integration: 34})
	if err != nil {
		t.Fatalf("updateTestsSection returned error: %v", err)
	}
	if want := `unit_tests-12`; !contains(got, want) {
		t.Fatalf("updated body missing %q", want)
	}
	if want := `integration_tests-34`; !contains(got, want) {
		t.Fatalf("updated body missing %q", want)
	}
}

func TestExistingCountsFromReadme(t *testing.T) {
	body := `unit_tests-56 xxx integration_tests-78`

	counts, err := existingCountsFromReadme(body)
	if err != nil {
		t.Fatalf("existingCountsFromReadme returned error: %v", err)
	}
	if counts.Unit != 56 || counts.Integration != 78 {
		t.Fatalf("counts = %+v, want unit=56 integration=78", counts)
	}
}

func contains(s, needle string) bool {
	return len(s) >= len(needle) && index(s, needle) >= 0
}

func index(s, needle string) int {
	for i := 0; i+len(needle) <= len(s); i++ {
		if s[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
