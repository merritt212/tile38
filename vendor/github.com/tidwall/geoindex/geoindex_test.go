package geoindex

// These tests are ripped off from github.com/tidwall/rbang

import (
	"io/ioutil"
	"math/rand"
	"sort"
	"testing"
	"time"

	"github.com/tidwall/cities"
	"github.com/tidwall/rbang"
)

const dims = 2

type tBox struct {
	min [dims]float64
	max [dims]float64
}

var boxes []tBox
var points []tBox

func init() {
	seed := time.Now().UnixNano()
	// seed = 1532132365683340889
	println("seed:", seed)
	rand.Seed(seed)
}

func randPoints(N int) []tBox {
	boxes := make([]tBox, N)
	for i := 0; i < N; i++ {
		boxes[i].min[0] = rand.Float64()*360 - 180
		boxes[i].min[1] = rand.Float64()*180 - 90
		for j := 2; j < dims; j++ {
			boxes[i].min[j] = rand.Float64()
		}
		boxes[i].max = boxes[i].min
	}
	return boxes
}

func randBoxes(N int) []tBox {
	boxes := make([]tBox, N)
	for i := 0; i < N; i++ {
		boxes[i].min[0] = rand.Float64()*360 - 180
		boxes[i].min[1] = rand.Float64()*180 - 90
		for j := 2; j < dims; j++ {
			boxes[i].min[j] = rand.Float64() * 100
		}
		boxes[i].max[0] = boxes[i].min[0] + rand.Float64()
		boxes[i].max[1] = boxes[i].min[1] + rand.Float64()
		for j := 2; j < dims; j++ {
			boxes[i].max[j] = boxes[i].min[j] + rand.Float64()
		}
		if boxes[i].max[0] > 180 || boxes[i].max[1] > 90 {
			i--
		}
	}
	return boxes
}

func sortBoxes(boxes []tBox) {
	sort.Slice(boxes, func(i, j int) bool {
		for k := 0; k < len(boxes[i].min); k++ {
			if boxes[i].min[k] < boxes[j].min[k] {
				return true
			}
			if boxes[i].min[k] > boxes[j].min[k] {
				return false
			}
			if boxes[i].max[k] < boxes[j].max[k] {
				return true
			}
			if boxes[i].max[k] > boxes[j].max[k] {
				return false
			}
		}
		return i < j
	})
}

func sortBoxesNearby(boxes []tBox, min, max [2]float64) {
	sort.Slice(boxes, func(i, j int) bool {
		return testBoxDist(boxes[i].min, boxes[i].max, min, max) <
			testBoxDist(boxes[j].min, boxes[j].max, min, max)
	})
}

func testBoxDist(amin, amax, bmin, bmax [2]float64) float64 {
	var dist float64
	for i := 0; i < len(amin); i++ {
		var min, max float64
		if amin[i] > bmin[i] {
			min = amin[i]
		} else {
			min = bmin[i]
		}
		if amax[i] < bmax[i] {
			max = amax[i]
		} else {
			max = bmax[i]
		}
		squared := min - max
		if squared > 0 {
			dist += squared * squared
		}
	}
	return dist
}

func testBoxesVarious(t *testing.T, boxes []tBox, label string) {
	N := len(boxes)

	tr := Wrap(&rbang.RTree{})

	// N := 10000
	// boxes := randPoints(N)

	/////////////////////////////////////////
	// insert
	/////////////////////////////////////////
	for i := 0; i < N; i++ {
		tr.Insert(boxes[i].min, boxes[i].max, boxes[i])
	}
	if tr.Len() != N {
		t.Fatalf("expected %d, got %d", N, tr.Len())
	}
	// area := tr.TotalOverlapArea()
	// fmt.Printf("overlap:    %.0f, %.1f/item\n", area, area/float64(N))

	//	ioutil.WriteFile(label+".svg", []byte(rtreetools.SVG(&tr)), 0600)

	/////////////////////////////////////////
	// scan all items and count one-by-one
	/////////////////////////////////////////
	var count int
	tr.Scan(func(min, max [2]float64, value interface{}) bool {
		count++
		return true
	})
	if count != N {
		t.Fatalf("expected %d, got %d", N, count)
	}

	/////////////////////////////////////////
	// check every point for correctness
	/////////////////////////////////////////
	var tboxes1 []tBox
	tr.Scan(func(min, max [2]float64, value interface{}) bool {
		tboxes1 = append(tboxes1, value.(tBox))
		return true
	})
	tboxes2 := make([]tBox, len(boxes))
	copy(tboxes2, boxes)
	sortBoxes(tboxes1)
	sortBoxes(tboxes2)
	for i := 0; i < len(tboxes1); i++ {
		if tboxes1[i] != tboxes2[i] {
			t.Fatalf("expected '%v', got '%v'", tboxes2[i], tboxes1[i])
		}
	}

	/////////////////////////////////////////
	// search for each item one-by-one
	/////////////////////////////////////////
	for i := 0; i < N; i++ {
		var found bool
		tr.Search(boxes[i].min, boxes[i].max,
			func(min, max [2]float64, value interface{}) bool {
				if value == boxes[i] {
					found = true
					return false
				}
				return true
			})
		if !found {
			t.Fatalf("did not find item %d", i)
		}
	}

	centerMin, centerMax := [2]float64{-18, -9}, [2]float64{18, 9}

	/////////////////////////////////////////
	// search for 10% of the items
	/////////////////////////////////////////
	for i := 0; i < N/5; i++ {
		var count int
		tr.Search(centerMin, centerMax,
			func(min, max [2]float64, value interface{}) bool {
				count++
				return true
			},
		)
	}

	/////////////////////////////////////////
	// delete every other item
	/////////////////////////////////////////
	for i := 0; i < N/2; i++ {
		j := i * 2
		tr.Delete(boxes[j].min, boxes[j].max, boxes[j])
	}

	/////////////////////////////////////////
	// count all items. should be half of N
	/////////////////////////////////////////
	count = 0
	tr.Scan(func(min, max [2]float64, value interface{}) bool {
		count++
		return true
	})
	if count != N/2 {
		t.Fatalf("expected %d, got %d", N/2, count)
	}

	///////////////////////////////////////////////////
	// reinsert every other item, but in random order
	///////////////////////////////////////////////////
	var ij []int
	for i := 0; i < N/2; i++ {
		j := i * 2
		ij = append(ij, j)
	}
	rand.Shuffle(len(ij), func(i, j int) {
		ij[i], ij[j] = ij[j], ij[i]
	})
	for i := 0; i < N/2; i++ {
		j := ij[i]
		tr.Insert(boxes[j].min, boxes[j].max, boxes[j])
	}

	//////////////////////////////////////////////////////
	// replace each item with an item that is very close
	//////////////////////////////////////////////////////
	var nboxes = make([]tBox, N)
	for i := 0; i < N; i++ {
		for j := 0; j < len(boxes[i].min); j++ {
			nboxes[i].min[j] = boxes[i].min[j] + (rand.Float64() - 0.5)
			if boxes[i].min == boxes[i].max {
				nboxes[i].max[j] = nboxes[i].min[j]
			} else {
				nboxes[i].max[j] = boxes[i].max[j] + (rand.Float64() - 0.5)
			}
		}

	}
	for i := 0; i < N; i++ {
		tr.Insert(nboxes[i].min, nboxes[i].max, nboxes[i])
		tr.Delete(boxes[i].min, boxes[i].max, boxes[i])
	}
	if tr.Len() != N {
		t.Fatalf("expected %d, got %d", N, tr.Len())
	}
	// area = tr.TotalOverlapArea()
	// fmt.Fprintf(wr, "overlap:    %.0f, %.1f/item\n", area, area/float64(N))

	/////////////////////////////////////////
	// check every point for correctness
	/////////////////////////////////////////
	tboxes1 = nil
	tr.Scan(func(min, max [2]float64, value interface{}) bool {
		tboxes1 = append(tboxes1, value.(tBox))
		return true
	})
	tboxes2 = make([]tBox, len(nboxes))
	copy(tboxes2, nboxes)
	sortBoxes(tboxes1)
	sortBoxes(tboxes2)
	for i := 0; i < len(tboxes1); i++ {
		if tboxes1[i] != tboxes2[i] {
			t.Fatalf("expected '%v', got '%v'", tboxes2[i], tboxes1[i])
		}
	}

	/////////////////////////////////////////
	// search for 10% of the items
	/////////////////////////////////////////
	for i := 0; i < N/5; i++ {
		var count int
		tr.Search(centerMin, centerMax,
			func(min, max [2]float64, value interface{}) bool {
				count++
				return true
			},
		)
	}

	var boxes3 []tBox
	tr.Nearby(
		SimpleBoxAlgo(centerMin, centerMax),
		func(min, max [2]float64, value interface{}, dist float64) bool {
			boxes3 = append(boxes3, value.(tBox))
			return true
		},
	)

	if len(boxes3) != len(nboxes) {
		t.Fatalf("expected %d, got %d", len(nboxes), len(boxes3))
	}
	if len(boxes3) != tr.Len() {
		t.Fatalf("expected %d, got %d", tr.Len(), len(boxes3))
	}

	var ldist float64
	for i, box := range boxes3 {
		dist := testBoxDist(box.min, box.max, centerMin, centerMax)
		if i > 0 && dist < ldist {
			t.Fatalf("out of order")
		}
		ldist = dist
	}

}

func TestRandomBoxes(t *testing.T) {
	testBoxesVarious(t, randBoxes(10000), "boxes")
}

func TestRandomPoints(t *testing.T) {
	testBoxesVarious(t, randPoints(10000), "points")
}

func TestSVG(t *testing.T) {
	var tr rbang.RTree
	index := Wrap(&tr)
	for _, city := range cities.Cities {
		p := [2]float64{city.Longitude, city.Latitude}
		tr.Insert(p, p, &city)
	}
	svg := index.SVG()
	if false {
		ioutil.WriteFile("cities.svg", []byte(svg), 0600)
	}
}
