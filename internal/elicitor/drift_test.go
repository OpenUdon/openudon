package elicitor

import "testing"

func TestTextMeaningfullyDiffersRequiresMoreThanOneSharedToken(t *testing.T) {
	if !textMeaningfullyDiffers("Fetch ticket", "Close ticket and notify finance") {
		t.Fatal("single shared token masked meaningful drift")
	}
	if textMeaningfullyDiffers("Fetch support ticket", "Fetch the support ticket") {
		t.Fatal("equivalent token set was treated as drift")
	}
	if textMeaningfullyDiffers("", "Fetch ticket") {
		t.Fatal("empty project text should stay non-blocking")
	}
}
