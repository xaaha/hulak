package tui

import "testing"

func TestDistributeSpaceEmpty(t *testing.T) {
	got := DistributeSpace(100, nil)
	if got != nil {
		t.Errorf("expected nil for empty entries, got %v", got)
	}
}

func TestDistributeSpaceSingleEntry(t *testing.T) {
	got := DistributeSpace(80, []LayoutEntry{{Weight: 1, MinSize: 10}})
	if len(got) != 1 || got[0] != 80 {
		t.Errorf("single entry should get all space, got %v", got)
	}
}

func TestDistributeSpaceEqualWeights(t *testing.T) {
	got := DistributeSpace(100, []LayoutEntry{
		{Weight: 1, MinSize: 0},
		{Weight: 1, MinSize: 0},
	})
	if got[0] != 50 || got[1] != 50 {
		t.Errorf("equal weights should split evenly, got %v", got)
	}
}

func TestDistributeSpaceUnequalWeights(t *testing.T) {
	got := DistributeSpace(100, []LayoutEntry{
		{Weight: 3, MinSize: 0},
		{Weight: 1, MinSize: 0},
	})
	if got[0] != 75 || got[1] != 25 {
		t.Errorf("3:1 weights over 100 should be [75,25], got %v", got)
	}
}

func TestDistributeSpaceMinSizeRespected(t *testing.T) {
	got := DistributeSpace(100, []LayoutEntry{
		{Weight: 1, MinSize: 40},
		{Weight: 1, MinSize: 40},
	})
	if got[0] < 40 || got[1] < 40 {
		t.Errorf("both entries should be >= 40, got %v", got)
	}
	if got[0]+got[1] != 100 {
		t.Errorf("sizes should sum to 100, got %d", got[0]+got[1])
	}
}

func TestDistributeSpaceTotalBelowMinimums(t *testing.T) {
	got := DistributeSpace(30, []LayoutEntry{
		{Weight: 1, MinSize: 20},
		{Weight: 1, MinSize: 20},
	})
	if got[0] != 20 || got[1] != 20 {
		t.Errorf("when total < minimums, each entry should still get MinSize, got %v", got)
	}
}

func TestDistributeSpaceRoundingRemainder(t *testing.T) {
	got := DistributeSpace(100, []LayoutEntry{
		{Weight: 1, MinSize: 0},
		{Weight: 1, MinSize: 0},
		{Weight: 1, MinSize: 0},
	})
	sum := got[0] + got[1] + got[2]
	if sum != 100 {
		t.Errorf("sizes should sum to 100 even with rounding, got %d (%v)", sum, got)
	}
	if got[0] < got[2] {
		t.Errorf("rounding remainder should go to earlier entries, got %v", got)
	}
}

func TestDistributeSpaceZeroWeight(t *testing.T) {
	got := DistributeSpace(100, []LayoutEntry{
		{Weight: 0, MinSize: 10},
		{Weight: 0, MinSize: 20},
	})
	if got[0] != 10 || got[1] != 20 {
		t.Errorf("zero weight entries should get only MinSize, got %v", got)
	}
}

func TestDistributeSpaceThreePanels(t *testing.T) {
	got := DistributeSpace(120, []LayoutEntry{
		{Weight: 2, MinSize: 10},
		{Weight: 1, MinSize: 10},
		{Weight: 1, MinSize: 10},
	})
	sum := got[0] + got[1] + got[2]
	if sum != 120 {
		t.Errorf("sizes should sum to 120, got %d (%v)", sum, got)
	}
	if got[0] <= got[1] {
		t.Errorf("first entry (weight 2) should be larger than second (weight 1), got %v", got)
	}
}
