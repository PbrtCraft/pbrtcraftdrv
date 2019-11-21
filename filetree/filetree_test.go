package filetree

import (
	"testing"
)

func TestGetFolder(t *testing.T) {
	ft, err := GetFolder(".")
	if err != nil {
		t.Fail()
		return
	}
	if len(ft.Files) != 2 {
		t.Fail()
	}
}
