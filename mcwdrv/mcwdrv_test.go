package mcwdrv

import "testing"

func TestGetPbrtStatus(t *testing.T) {
	ps, err := parsePbrtStatus("Rendering: [+++++++++++++++++++++++                    ]  (1.0s|0.9s)")
	if err != nil {
		t.Error(err)
	}

	if ps.AllSec != 1.0 || ps.LeaveSec != 0.9 {
		t.Fail()
	}
}
