package game

import (
	"math"
	"sort"
)

type treeSpec struct {
	X, Y   int
	Radius int
}

type rectI struct{ X0, Y0, X1, Y1 int }

type buildingSpec struct {
	Col        RGB
	Outline    RGB
	H          uint8
	Parts      []rectI
	RoofRim    bool
	RimAddH    uint8
	RoofUnits  []rectI
	UnitAddH   uint8
	HasYard    bool
	ExtraTrees bool
}

type blockFeatures struct {
	Seed      uint64
	Parcel    blockParcel
	IsParking bool
	IsPark    bool
	ParkRects []rectI
	Buildings []buildingSpec
	Trees     []treeSpec
}

type themePalette struct {
	Road         RGB
	Sidewalk     RGB
	Lot          RGB
	BuildingA    RGB
	BuildingB    RGB
	BuildingC    RGB
	BuildingDark RGB
	Grass        RGB
	GrassPatch   RGB
	GrassTorn    RGB
	TreeBase     RGB
	TreeMid      RGB
	TreeTop      RGB
}

func roadStripeColor(tp themePalette) RGB {
	return tp.Sidewalk.Add(88, 88, 84)
}

func isRoadSurfaceColor(col RGB, tp themePalette) bool {
	return rgbEq(col, tp.Road) || rgbEq(col, roadStripeColor(tp))
}

func chunkColorAt(c *Chunk, idx int) RGB {
	o := c.pixOff(idx)
	return RGB{R: c.Pixels[o+0], G: c.Pixels[o+1], B: c.Pixels[o+2]}
}

func buildThemePalette(theme ThemeConfig) themePalette {
	tp := themePalette{
		Road:         Palette.Road,
		Sidewalk:     Palette.Sidewalk,
		Lot:          Palette.Lot,
		BuildingA:    Palette.BuildingA,
		BuildingB:    Palette.BuildingB,
		BuildingC:    Palette.BuildingC,
		BuildingDark: Palette.BuildingDark,
		Grass:        Palette.Grass,
		GrassPatch:   Palette.GrassPatch,
		GrassTorn:    Palette.GrassTorn,
		TreeBase:     Palette.TreeBase,
		TreeMid:      Palette.TreeMid,
		TreeTop:      Palette.TreeTop,
	}
	shiftTerrain := func(dr, dg, db int) {
		tp.Road = tp.Road.Add(dr, dg, db)
		tp.Sidewalk = tp.Sidewalk.Add(dr, dg, db)
		tp.Lot = tp.Lot.Add(dr, dg, db)
	}
	shiftBuildings := func(dr, dg, db int) {
		tp.BuildingA = tp.BuildingA.Add(dr, dg, db)
		tp.BuildingB = tp.BuildingB.Add(dr, dg, db)
		tp.BuildingC = tp.BuildingC.Add(dr, dg, db)
		tp.BuildingDark = tp.BuildingDark.Add(dr/2, dg/2, db/2)
	}
	shiftGreen := func(dr, dg, db int) {
		tp.Grass = tp.Grass.Add(dr, dg, db)
		tp.GrassPatch = tp.GrassPatch.Add(dr, dg, db)
		tp.GrassTorn = tp.GrassTorn.Add(dr, dg, db)
		tp.TreeBase = tp.TreeBase.Add(dr, dg, db)
		tp.TreeMid = tp.TreeMid.Add(dr, dg, db)
		tp.TreeTop = tp.TreeTop.Add(dr, dg, db)
	}

	switch theme.FamilyName() {
	case ThemeSuburban.Name:
		shiftTerrain(8, 6, 2)
		shiftBuildings(10, 8, 4)
		shiftGreen(6, 12, 4)
	case ThemeForest.Name:
		shiftTerrain(-14, -10, -6)
		shiftBuildings(-18, -12, -8)
		shiftGreen(-6, 22, -4)
	case ThemeForestSpring.Name:
		shiftTerrain(2, 2, 0)
		shiftBuildings(-8, -2, -4)
		shiftGreen(10, 28, 8)
		tp.Grass = RGB{R: 150, G: 174, B: 112}
		tp.GrassPatch = RGB{R: 132, G: 158, B: 98}
		tp.GrassTorn = RGB{R: 170, G: 190, B: 126}
		tp.TreeBase = RGB{R: 64, G: 108, B: 58}
		tp.TreeMid = RGB{R: 92, G: 148, B: 74}
		tp.TreeTop = RGB{R: 136, G: 190, B: 98}
	case ThemeForestSummer.Name:
		shiftTerrain(-10, -8, -2)
		shiftBuildings(-14, -8, -6)
		shiftGreen(-2, 32, -2)
		tp.Grass = RGB{R: 118, G: 144, B: 90}
		tp.GrassPatch = RGB{R: 100, G: 126, B: 76}
		tp.GrassTorn = RGB{R: 140, G: 162, B: 98}
		tp.TreeBase = RGB{R: 48, G: 82, B: 40}
		tp.TreeMid = RGB{R: 70, G: 110, B: 52}
		tp.TreeTop = RGB{R: 98, G: 142, B: 64}
	case ThemeForestAutumn.Name:
		shiftTerrain(12, 4, -8)
		shiftBuildings(6, 2, -6)
		shiftGreen(34, -8, -26)
		tp.Grass = RGB{R: 166, G: 142, B: 86}
		tp.GrassPatch = RGB{R: 146, G: 122, B: 72}
		tp.GrassTorn = RGB{R: 188, G: 162, B: 96}
		tp.TreeBase = RGB{R: 98, G: 72, B: 40}
		tp.TreeMid = RGB{R: 162, G: 106, B: 48}
		tp.TreeTop = RGB{R: 214, G: 146, B: 58}
	case ThemeForestWinter.Name:
		shiftTerrain(-16, -6, 20)
		shiftBuildings(-10, -4, 18)
		shiftGreen(-28, -8, 30)
		tp.Lot = RGB{R: 198, G: 206, B: 214}
		tp.Grass = RGB{R: 214, G: 222, B: 230}
		tp.GrassPatch = RGB{R: 198, G: 206, B: 214}
		tp.GrassTorn = RGB{R: 228, G: 234, B: 240}
		tp.TreeBase = RGB{R: 76, G: 88, B: 88}
		tp.TreeMid = RGB{R: 136, G: 148, B: 154}
		tp.TreeTop = RGB{R: 194, G: 208, B: 216}
	case ThemeRural.Name:
		shiftTerrain(12, 8, 2)
		shiftBuildings(16, 10, 2)
		shiftGreen(8, 14, 0)
	case ThemeArctic.Name:
		shiftTerrain(-18, -8, 24)
		shiftBuildings(-10, -4, 28)
		shiftGreen(-25, -10, 35)
	case ThemeDesert.Name:
		shiftTerrain(30, 10, -18)
		shiftBuildings(40, 12, -20)
		shiftGreen(20, 4, -30)
	case ThemeSand.Name:
		shiftTerrain(34, 16, -14)
		shiftBuildings(36, 16, -16)
		shiftGreen(28, 10, -24)
	case ThemeWinter.Name:
		shiftTerrain(-20, -10, 24)
		shiftBuildings(-12, -6, 22)
		shiftGreen(-24, -10, 30)
		tp.Lot = RGB{R: 186, G: 196, B: 206}
		tp.Grass = RGB{R: 206, G: 214, B: 224}
		tp.GrassPatch = RGB{R: 188, G: 198, B: 208}
		tp.GrassTorn = RGB{R: 222, G: 228, B: 236}
		tp.TreeBase = RGB{R: 84, G: 96, B: 96}
		tp.TreeMid = RGB{R: 146, G: 160, B: 164}
		tp.TreeTop = RGB{R: 204, G: 218, B: 224}
	case ThemeBeach.Name:
		shiftTerrain(28, 14, 4)
		shiftBuildings(18, 10, 8)
		shiftGreen(6, 18, 16)
	case ThemeSpace.Name:
		shiftTerrain(-22, -24, 26)
		shiftBuildings(-18, -24, 30)
		shiftGreen(-28, -20, 16)
	case ThemeUnderwater.Name:
		shiftTerrain(-22, 4, 22)
		shiftBuildings(-24, 0, 26)
		shiftGreen(-16, 12, 24)
	case ThemeMegacity.Name:
		shiftTerrain(-20, -16, -12)
		shiftBuildings(-18, -14, -10)
		shiftGreen(-20, -12, -10)
	case ThemeParkCity.Name:
		shiftTerrain(10, 8, 4)
		shiftBuildings(4, 12, 2)
		shiftGreen(10, 24, 4)
	case ThemeVillage.Name:
		shiftTerrain(12, 8, 4)
		shiftBuildings(20, 10, 0)
		shiftGreen(8, 16, 2)
	case ThemeIndustrial.Name:
		shiftTerrain(-18, -16, -10)
		shiftBuildings(-24, -18, -12)
		shiftGreen(-16, -6, -8)
	case ThemeWasteland.Name:
		shiftTerrain(8, -12, -16)
		shiftBuildings(6, -16, -18)
		shiftGreen(-10, -18, -20)
	case ThemeJungle.Name:
		shiftTerrain(-10, 0, -8)
		shiftBuildings(-18, -4, -12)
		shiftGreen(-8, 30, -6)
	case ThemeCanyon.Name:
		shiftTerrain(24, 4, -18)
		shiftBuildings(30, 6, -20)
		shiftGreen(6, -6, -22)
	case ThemeVolcanic.Name:
		shiftTerrain(8, -22, -20)
		shiftBuildings(14, -24, -18)
		shiftGreen(-12, -24, -24)
	case ThemeSwamp.Name:
		shiftTerrain(-8, 4, -10)
		shiftBuildings(-6, 2, -8)
		shiftGreen(-18, 18, -12)
	case ThemeNeon.Name:
		shiftTerrain(0, -10, 24)
		shiftBuildings(16, -12, 34)
		shiftGreen(-12, 6, 20)
	case ThemeRuins.Name:
		shiftTerrain(2, -8, -8)
		shiftBuildings(-4, -10, -10)
		shiftGreen(-6, 8, -4)
	case ThemeFarmland.Name:
		shiftTerrain(14, 10, 2)
		shiftBuildings(18, 12, 4)
		shiftGreen(14, 22, 0)
	case ThemeHighlands.Name:
		shiftTerrain(-4, -2, 12)
		shiftBuildings(-4, 0, 10)
		shiftGreen(-2, 8, 10)
	}
	return tp
}

func isWinterTheme(theme ThemeConfig) bool {
	switch theme.FamilyName() {
	case ThemeArctic.Name, ThemeWinter.Name, ThemeForestWinter.Name:
		return true
	default:
		return false
	}
}

// isPerimeterRoad reports whether a world pixel is in the ring road
// just inside the unbreakable border.
func isPerimeterRoad(wx, wy int) bool {
	if wx < BorderThickness || wy < BorderThickness || wx >= WorldWidth-BorderThickness || wy >= WorldHeight-BorderThickness {
		return false
	}
	return wx < BorderThickness+RoadWidth ||
		wy < BorderThickness+RoadWidth ||
		wx >= WorldWidth-BorderThickness-RoadWidth ||
		wy >= WorldHeight-BorderThickness-RoadWidth
}

func roadStripeDash(pos int) bool {
	return (pos % 7) < 2 // slightly longer dashes than before
}

func perimeterRoadColor(wx, wy int, tp themePalette) RGB {
	col := tp.Road
	topCenter := BorderThickness + RoadWidth/2
	bottomCenter := WorldHeight - BorderThickness - RoadWidth + RoadWidth/2
	leftCenter := BorderThickness + RoadWidth/2
	rightCenter := WorldWidth - BorderThickness - RoadWidth + RoadWidth/2

	// In corner overlap zones, use one axis only so stripes stay straight.
	dTop := wy - BorderThickness
	dBottom := (WorldHeight - BorderThickness - 1) - wy
	dLeft := wx - BorderThickness
	dRight := (WorldWidth - BorderThickness - 1) - wx
	hBandDist := min(dTop, dBottom)
	vBandDist := min(dLeft, dRight)

	useHorizontal := hBandDist <= vBandDist
	if useHorizontal {
		if (wy == topCenter || wy == bottomCenter) && roadStripeDash(wx) {
			col = roadStripeColor(tp)
		}
	} else if (wx == leftCenter || wx == rightCenter) && roadStripeDash(wy) {
		col = roadStripeColor(tp)
	}
	return col
}

// generateChunk fills a chunk with deterministic city content.
func generateChunk(c *Chunk, worldSeed uint64, theme ThemeConfig) {
	tp := buildThemePalette(theme)
	baseX, baseY := c.WorldOrigin()
	maxX := baseX + ChunkSize - 1
	maxY := baseY + ChunkSize - 1
	maxBX := floorDiv(WorldWidth-1, Pattern)
	maxBY := floorDiv(WorldHeight-1, Pattern)

	parcelCache := make(map[int64]blockParcel, 64)
	pKey := func(bx, by int) int64 {
		return (int64(by) << 32) ^ int64(uint32(bx))
	}
	getParcel := func(bx, by int) blockParcel {
		if bx < 0 || by < 0 || bx > maxBX || by > maxBY {
			return blockParcel{Kind: parcelNone}
		}
		key := pKey(bx, by)
		if p, ok := parcelCache[key]; ok {
			return p
		}
		profile := blockProfileFor(worldSeed, bx, by)
		p := blockParcelFor(worldSeed, bx, by, theme, profile)
		parcelCache[key] = p
		return p
	}
	sharedParcelKind := func(ax, ay, bx, by int) parcelKind {
		pa := getParcel(ax, ay)
		pb := getParcel(bx, by)
		if pa.Kind == parcelNone || pb.Kind == parcelNone || pa.Kind != pb.Kind {
			return parcelNone
		}
		if pa.AX != pb.AX || pa.AY != pb.AY || pa.W != pb.W || pa.H != pb.H {
			return parcelNone
		}
		return pa.Kind
	}

	gridW := maxBX + 1
	gridH := maxBY + 1
	vertRoadOpen := make([][]bool, gridW+1)
	for x := 0; x <= gridW; x++ {
		vertRoadOpen[x] = make([]bool, gridH)
		for y := 0; y < gridH; y++ {
			vertRoadOpen[x][y] = true
		}
	}
	horzRoadOpen := make([][]bool, gridW)
	for x := 0; x < gridW; x++ {
		horzRoadOpen[x] = make([]bool, gridH+1)
		for y := 0; y <= gridH; y++ {
			horzRoadOpen[x][y] = true
		}
	}

	type roadCloseCandidate struct {
		Vertical bool
		K, C     int
		Score    uint64
	}
	candidates := make([]roadCloseCandidate, 0, (gridW-1)*gridH+gridW*(gridH-1))
	for y := 0; y < gridH; y++ {
		for k := 1; k < gridW; k++ {
			if sharedParcelKind(k-1, y, k, y) == parcelNone {
				continue
			}
			candidates = append(candidates, roadCloseCandidate{
				Vertical: true,
				K:        k,
				C:        y,
				Score:    hash2D(worldSeed^0xA1B2C3D4E5F60718, k, y),
			})
		}
	}
	for x := 0; x < gridW; x++ {
		for k := 1; k < gridH; k++ {
			if sharedParcelKind(x, k-1, x, k) == parcelNone {
				continue
			}
			candidates = append(candidates, roadCloseCandidate{
				Vertical: false,
				K:        k,
				C:        x,
				Score:    hash2D(worldSeed^0x1029384756ABCDEF, x, k),
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].Score > candidates[j].Score })

	nodeID := func(ix, iy int) int { return iy*(gridW+1) + ix }
	nodeDegree := func(ix, iy int) int {
		d := 0
		if iy > 0 && vertRoadOpen[ix][iy-1] {
			d++
		}
		if iy < gridH && vertRoadOpen[ix][iy] {
			d++
		}
		if ix > 0 && horzRoadOpen[ix-1][iy] {
			d++
		}
		if ix < gridW && horzRoadOpen[ix][iy] {
			d++
		}
		return d
	}
	hasLooseEnds := func() bool {
		for iy := 0; iy <= gridH; iy++ {
			for ix := 0; ix <= gridW; ix++ {
				if nodeDegree(ix, iy) == 1 {
					return true
				}
			}
		}
		return false
	}
	allRoadsConnected := func() bool {
		total := (gridW + 1) * (gridH + 1)
		vis := make([]bool, total)
		q := make([]int, 0, total)
		start := 0
		vis[start] = true
		q = append(q, start)
		for qi := 0; qi < len(q); qi++ {
			n := q[qi]
			x := n % (gridW + 1)
			y := n / (gridW + 1)
			if y > 0 && vertRoadOpen[x][y-1] {
				nn := nodeID(x, y-1)
				if !vis[nn] {
					vis[nn] = true
					q = append(q, nn)
				}
			}
			if y < gridH && vertRoadOpen[x][y] {
				nn := nodeID(x, y+1)
				if !vis[nn] {
					vis[nn] = true
					q = append(q, nn)
				}
			}
			if x > 0 && horzRoadOpen[x-1][y] {
				nn := nodeID(x-1, y)
				if !vis[nn] {
					vis[nn] = true
					q = append(q, nn)
				}
			}
			if x < gridW && horzRoadOpen[x][y] {
				nn := nodeID(x+1, y)
				if !vis[nn] {
					vis[nn] = true
					q = append(q, nn)
				}
			}
		}
		for i := range vis {
			if !vis[i] {
				return false
			}
		}
		return true
	}
	for _, cand := range candidates {
		if cand.Vertical {
			if !vertRoadOpen[cand.K][cand.C] {
				continue
			}
			vertRoadOpen[cand.K][cand.C] = false
			if !allRoadsConnected() || hasLooseEnds() {
				vertRoadOpen[cand.K][cand.C] = true
			}
		} else {
			if !horzRoadOpen[cand.C][cand.K] {
				continue
			}
			horzRoadOpen[cand.C][cand.K] = false
			if !allRoadsConnected() || hasLooseEnds() {
				horzRoadOpen[cand.C][cand.K] = true
			}
		}
	}

	// Base classification: border/road/sidewalk/lot (or all grass when NoRoads).
	for y := 0; y < ChunkSize; y++ {
		wy := baseY + y
		for x := 0; x < ChunkSize; x++ {
			wx := baseX + x
			i := c.idx(x, y)

			if wx < 0 || wy < 0 || wx >= WorldWidth || wy >= WorldHeight {
				c.set(i, Palette.Border, BorderHeight, ShadeLit, 1)
				continue
			}
			if wx < BorderThickness || wy < BorderThickness || wx >= WorldWidth-BorderThickness || wy >= WorldHeight-BorderThickness {
				c.set(i, Palette.Border, BorderHeight, ShadeLit, 1)
				continue
			}
			if isPerimeterRoad(wx, wy) {
				c.set(i, perimeterRoadColor(wx, wy, tp), 0, ShadeLit, 0)
				continue
			}

			if theme.NoRoads {
				col := tp.Grass
				pv := uint8(hash2D(worldSeed^0xBADC0FFEE0DDF00D, wx>>4, wy>>4) >> 56)
				if pv < 18 {
					col = tp.GrassTorn
				} else if pv < 60 {
					col = tp.GrassPatch
				}
				hv := uint8(hash2D(worldSeed^0x1234567, wx, wy) >> 56)
				switch (wx + 2*wy + int(hv&3)) & 3 {
				case 1:
					col = col.Add(6, 6, 4)
				case 2:
					col = col.Add(-6, -6, -4)
				}
				c.set(i, col, 0, ShadeLit, 0)
				continue
			}

			xm := wx % Pattern
			ym := wy % Pattern
			xRoad := xm < RoadWidth
			yRoad := ym < RoadWidth
			xSide := (xm >= RoadWidth && xm < RoadWidth+SidewalkWidth) || (xm >= RoadWidth+SidewalkWidth+BlockInner)
			ySide := (ym >= RoadWidth && ym < RoadWidth+SidewalkWidth) || (ym >= RoadWidth+SidewalkWidth+BlockInner)
			cellX := floorDiv(wx, Pattern)
			cellY := floorDiv(wy, Pattern)

			verticalKind := parcelNone
			xClosed := false
			if xRoad || xSide {
				if xm < RoadWidth+SidewalkWidth {
					k := cellX
					if k > 0 && k < gridW && cellY >= 0 && cellY < gridH && !vertRoadOpen[k][cellY] {
						verticalKind = sharedParcelKind(cellX-1, cellY, cellX, cellY)
						xClosed = verticalKind != parcelNone
					}
				} else if xm >= RoadWidth+SidewalkWidth+BlockInner {
					k := cellX + 1
					if k > 0 && k < gridW && cellY >= 0 && cellY < gridH && !vertRoadOpen[k][cellY] {
						verticalKind = sharedParcelKind(cellX, cellY, cellX+1, cellY)
						xClosed = verticalKind != parcelNone
					}
				}
			}
			horizontalKind := parcelNone
			yClosed := false
			if yRoad || ySide {
				if ym < RoadWidth+SidewalkWidth {
					k := cellY
					if k > 0 && k < gridH && cellX >= 0 && cellX < gridW && !horzRoadOpen[cellX][k] {
						horizontalKind = sharedParcelKind(cellX, cellY-1, cellX, cellY)
						yClosed = horizontalKind != parcelNone
					}
				} else if ym >= RoadWidth+SidewalkWidth+BlockInner {
					k := cellY + 1
					if k > 0 && k < gridH && cellX >= 0 && cellX < gridW && !horzRoadOpen[cellX][k] {
						horizontalKind = sharedParcelKind(cellX, cellY, cellX, cellY+1)
						yClosed = horizontalKind != parcelNone
					}
				}
			}

			xRoadOpen := xRoad && !xClosed
			yRoadOpen := yRoad && !yClosed
			xSideOpen := xSide && !xClosed
			ySideOpen := ySide && !yClosed

			internalKind := parcelNone
			if xClosed && verticalKind != parcelNone {
				internalKind = verticalKind
			}
			if yClosed && horizontalKind != parcelNone {
				if internalKind == parcelNone {
					internalKind = horizontalKind
				} else if horizontalKind != internalKind {
					internalKind = parcelNone
				}
			}

			if (xRoadOpen || yRoadOpen) || (xSideOpen || ySideOpen) {
				col := tp.Road
				// Dotted center line for straight road segments.
				if xRoadOpen != yRoadOpen {
					stripe := false
					if xRoadOpen && xm == RoadWidth/2 {
						stripe = roadStripeDash(wy)
					}
					if yRoadOpen && ym == RoadWidth/2 {
						stripe = roadStripeDash(wx)
					}
					if stripe {
						col = roadStripeColor(tp)
					}
				}
				if xRoadOpen || yRoadOpen {
					c.set(i, col, 0, ShadeLit, 0)
				} else {
					c.set(i, tp.Sidewalk, 0, ShadeLit, 0)
				}
				continue
			}

			if internalKind != parcelNone && (xRoad || yRoad || xSide || ySide) {
				if internalKind == parcelPark {
					col := tp.GrassPatch
					hv := uint8(hash2D(worldSeed^0x88112233, wx, wy) >> 56)
					if (hv & 3) == 0 {
						col = tp.Grass
					} else if (hv & 7) == 0 {
						col = tp.GrassTorn
					}
					c.set(i, col, 0, ShadeLit, 0)
				} else {
					col := tp.Lot.Add(-4, -4, -3)
					if ((wx + wy) & 3) == 0 {
						col = tp.Lot.Add(2, 2, 1)
					}
					c.set(i, col, 0, ShadeLit, 0)
				}
				continue
			}

			c.set(i, tp.Lot, 0, ShadeLit, 0)
		}
	}

	// Overlay block features.
	bxMin := floorDiv(baseX, Pattern)
	byMin := floorDiv(baseY, Pattern)
	bxMax := floorDiv(maxX, Pattern)
	byMax := floorDiv(maxY, Pattern)

	for by := byMin; by <= byMax; by++ {
		for bx := bxMin; bx <= bxMax; bx++ {
			feat := genBlockFeatures(worldSeed, bx, by, theme, tp)
			applyBlockFeatures(c, bx, by, feat, tp, theme)
		}
	}

	// Final road pass: roads are authored last from the solved road graph so
	// parks/buildings/trees can never overwrite or disconnect them.
	if !theme.NoRoads {
		for y := 0; y < ChunkSize; y++ {
			wy := baseY + y
			for x := 0; x < ChunkSize; x++ {
				wx := baseX + x
				i := c.idx(x, y)
				if wx < BorderThickness || wy < BorderThickness || wx >= WorldWidth-BorderThickness || wy >= WorldHeight-BorderThickness {
					continue
				}
				if isPerimeterRoad(wx, wy) {
					c.set(i, perimeterRoadColor(wx, wy, tp), 0, ShadeLit, 0)
					continue
				}

				xm := wx % Pattern
				ym := wy % Pattern
				xRoad := xm < RoadWidth
				yRoad := ym < RoadWidth
				xSide := (xm >= RoadWidth && xm < RoadWidth+SidewalkWidth) || (xm >= RoadWidth+SidewalkWidth+BlockInner)
				ySide := (ym >= RoadWidth && ym < RoadWidth+SidewalkWidth) || (ym >= RoadWidth+SidewalkWidth+BlockInner)
				cellX := floorDiv(wx, Pattern)
				cellY := floorDiv(wy, Pattern)

				verticalKind := parcelNone
				xClosed := false
				if xRoad || xSide {
					if xm < RoadWidth+SidewalkWidth {
						k := cellX
						if k > 0 && k < gridW && cellY >= 0 && cellY < gridH && !vertRoadOpen[k][cellY] {
							verticalKind = sharedParcelKind(cellX-1, cellY, cellX, cellY)
							xClosed = verticalKind != parcelNone
						}
					} else if xm >= RoadWidth+SidewalkWidth+BlockInner {
						k := cellX + 1
						if k > 0 && k < gridW && cellY >= 0 && cellY < gridH && !vertRoadOpen[k][cellY] {
							verticalKind = sharedParcelKind(cellX, cellY, cellX+1, cellY)
							xClosed = verticalKind != parcelNone
						}
					}
				}
				horizontalKind := parcelNone
				yClosed := false
				if yRoad || ySide {
					if ym < RoadWidth+SidewalkWidth {
						k := cellY
						if k > 0 && k < gridH && cellX >= 0 && cellX < gridW && !horzRoadOpen[cellX][k] {
							horizontalKind = sharedParcelKind(cellX, cellY-1, cellX, cellY)
							yClosed = horizontalKind != parcelNone
						}
					} else if ym >= RoadWidth+SidewalkWidth+BlockInner {
						k := cellY + 1
						if k > 0 && k < gridH && cellX >= 0 && cellX < gridW && !horzRoadOpen[cellX][k] {
							horizontalKind = sharedParcelKind(cellX, cellY, cellX, cellY+1)
							yClosed = horizontalKind != parcelNone
						}
					}
				}

				xRoadOpen := xRoad && !xClosed
				yRoadOpen := yRoad && !yClosed
				xSideOpen := xSide && !xClosed
				ySideOpen := ySide && !yClosed

				switch {
				case xRoadOpen || yRoadOpen:
					col := tp.Road
					if xRoadOpen != yRoadOpen {
						stripe := false
						if xRoadOpen && xm == RoadWidth/2 {
							stripe = roadStripeDash(wy)
						}
						if yRoadOpen && ym == RoadWidth/2 {
							stripe = roadStripeDash(wx)
						}
						if stripe {
							col = roadStripeColor(tp)
						}
					}
					c.set(i, col, 0, ShadeLit, 0)
				case xSideOpen || ySideOpen:
					c.set(i, tp.Sidewalk, 0, ShadeLit, 0)
				}
			}
		}
	}

	c.NeedsShadow = true
	c.NeedsUpload = true
}

type blockProfile struct {
	ParkBias         int
	BuildingBias     int
	SizeBias         int
	MegaParkChance   int
	SuperBlockChance int
}

type parcelKind uint8

const (
	parcelNone parcelKind = iota
	parcelPark
	parcelBuilding
	parcelParking
)

type blockParcel struct {
	Kind   parcelKind
	AX, AY int
	W, H   int
	Score  uint64
}

func blockProfileFor(worldSeed uint64, bx, by int) blockProfile {
	macroSeed := hash2D(worldSeed^0x8C6F30D9A2B5E741, bx>>2, by>>2)
	microSeed := hash2D(worldSeed^0xA3D21F4EC7B8C905, bx>>1, by>>1)
	zone := int((macroSeed ^ (microSeed >> 7)) & 7)

	p := blockProfile{}
	switch zone {
	case 0, 1:
		// Broad greener neighborhoods.
		p = blockProfile{ParkBias: 26, BuildingBias: -2, SizeBias: -2, MegaParkChance: 34, SuperBlockChance: 6}
	case 2, 3:
		// Denser but still mixed neighborhoods.
		p = blockProfile{ParkBias: -6, BuildingBias: 2, SizeBias: -1, MegaParkChance: -6, SuperBlockChance: 16}
	case 4, 5:
		// Industrial/super-block neighborhoods.
		p = blockProfile{ParkBias: -14, BuildingBias: -1, SizeBias: 4, MegaParkChance: -16, SuperBlockChance: 52}
	default:
		// Transitional mixed neighborhoods.
		p = blockProfile{ParkBias: 6, BuildingBias: 0, SizeBias: 2, MegaParkChance: 12, SuperBlockChance: 28}
	}

	local := hash2D(worldSeed^0x4B1D9F73EE2A90C7, bx, by)
	p.ParkBias += int((local>>57)&0x07) - 3
	p.BuildingBias += int((local>>53)&0x07) - 3
	p.SizeBias += int((local>>49)&0x07) - 3
	return p
}

func themeSizeToBlock(v int) int {
	v = clamp(v, 8, 95)
	scaled := 8 + (v-8)*max(4, BlockInner-10)/87
	return clamp(scaled, 6, BlockInner-2)
}

func chooseParcelDims(r *Rand, minArea, maxArea, maxSpan int) (int, int) {
	if maxArea < 1 {
		return 1, 1
	}
	if minArea < 1 {
		minArea = 1
	}
	if minArea > maxArea {
		minArea = maxArea
	}
	if maxSpan < 1 {
		maxSpan = 1
	}
	for i := 0; i < 12; i++ {
		w := r.Range(1, maxSpan)
		h := r.Range(1, maxSpan)
		a := w * h
		if a >= minArea && a <= maxArea {
			return w, h
		}
	}
	// Fallback: stretch one axis deterministically.
	if maxArea <= 2 {
		return maxArea, 1
	}
	if r.Intn(2) == 0 {
		return 2, max(1, min(maxSpan, maxArea/2))
	}
	return max(1, min(maxSpan, maxArea/2)), 2
}

func parcelCovers(p blockParcel, bx, by int) bool {
	if p.Kind == parcelNone || p.W <= 0 || p.H <= 0 {
		return false
	}
	return bx >= p.AX && bx < p.AX+p.W && by >= p.AY && by < p.AY+p.H
}

func blockParcelFor(worldSeed uint64, bx, by int, theme ThemeConfig, profile blockProfile) blockParcel {
	parkChance := clamp(8+(theme.ParkChance[0]+theme.ParkChance[1])/3+profile.ParkBias/2, 4, 70)
	buildingChance := clamp(10+(100-theme.ParkChance[1])/4+profile.SuperBlockChance/4+profile.BuildingBias, 4, 65)
	parkingChance := clamp(5+(100-theme.ParkChance[1])/12+profile.SuperBlockChance/14, 3, 24)
	if theme.NoRoads {
		buildingChance /= 2
		parkingChance = 0
	}

	best := blockParcel{Kind: parcelNone}
	for ay := by - 2; ay <= by; ay++ {
		for ax := bx - 2; ax <= bx; ax++ {
			parkSeed := hash2D(worldSeed^0xC0FFEE77A11CE55D, ax, ay)
			pr := NewRand(parkSeed)
			if pr.Intn(100) < parkChance {
				pw, ph := chooseParcelDims(pr, 2, 4, 3) // 2-4 blocks.
				p := blockParcel{
					Kind:  parcelPark,
					AX:    ax,
					AY:    ay,
					W:     pw,
					H:     ph,
					Score: hash2D(parkSeed^0x1199AACC5500DD33, ax+pw, ay+ph),
				}
				if parcelCovers(p, bx, by) && (best.Kind == parcelNone || p.Score > best.Score) {
					best = p
				}
			}

			buildSeed := hash2D(worldSeed^0x55EE10BADC0FFEE1, ax, ay)
			br := NewRand(buildSeed)
			if br.Intn(100) < buildingChance {
				bw, bh := chooseParcelDims(br, 2, 4, 2) // 2-4 blocks.
				p := blockParcel{
					Kind:  parcelBuilding,
					AX:    ax,
					AY:    ay,
					W:     bw,
					H:     bh,
					Score: hash2D(buildSeed^0xAA33DD551177CC99, ax+bw, ay+bh),
				}
				if parcelCovers(p, bx, by) && (best.Kind == parcelNone || p.Score > best.Score) {
					best = p
				}
			}

			lotSeed := hash2D(worldSeed^0xF00DCCAA11773355, ax, ay)
			lr := NewRand(lotSeed)
			if lr.Intn(100) < parkingChance {
				p := blockParcel{
					Kind:  parcelParking,
					AX:    ax,
					AY:    ay,
					W:     1,
					H:     1,
					Score: hash2D(lotSeed^0x55AA7711CCDD0099, ax, ay),
				}
				if parcelCovers(p, bx, by) && (best.Kind == parcelNone || p.Score > best.Score) {
					best = p
				}
			}
		}
	}
	return best
}

func genBlockFeatures(worldSeed uint64, bx, by int, theme ThemeConfig, tp themePalette) blockFeatures {
	seed := hash2D(worldSeed, bx, by)
	r := NewRand(seed)
	profile := blockProfileFor(worldSeed, bx, by)
	parcel := blockParcelFor(worldSeed, bx, by, theme, profile)
	parcelArea := max(1, parcel.W*parcel.H)
	forcedPark := parcel.Kind == parcelPark
	forcedBuilding := parcel.Kind == parcelBuilding
	forcedParking := parcel.Kind == parcelParking
	if forcedParking {
		return blockFeatures{Seed: seed, Parcel: parcel, IsParking: true}
	}

	// Park chance varies by district, scaled by theme range.
	district := int(hash2D(worldSeed^0xD0D0D0D0D0D0D0D0, bx>>2, by>>2) >> 56)
	districtT := float64(district) / 255.0 // 0..1
	parkChance := theme.ParkChance[0] + int(districtT*float64(theme.ParkChance[1]-theme.ParkChance[0]))
	parkChance = clamp(parkChance+profile.ParkBias, 0, 95)
	if forcedPark {
		parkChance = 100
	}
	if forcedBuilding {
		parkChance = 0
	}

	// Parks.
	if forcedPark || r.Intn(100) < parkChance {
		megaChance := clamp(22+profile.MegaParkChance, 6, 90)
		if forcedPark {
			megaChance = clamp(35+parcelArea*12, 35, 95)
		}
		mega := r.Intn(100) < megaChance
		rectCount := 1
		if !mega {
			rectCount = 1 + r.Intn(3)
			if r.Intn(100) < 35 {
				rectCount++
			}
		}
		if forcedPark {
			rectCount = max(1, rectCount-1)
		}
		rects := make([]rectI, 0, rectCount)
		for i := 0; i < rectCount; i++ {
			if mega {
				m := r.Range(1, 4)
				wMin := max(BlockInner/2, BlockInner-m-6)
				hMin := max(BlockInner/2, BlockInner-m-6)
				wMax := max(wMin, BlockInner-m)
				hMax := max(hMin, BlockInner-m)
				w := r.Range(wMin, wMax)
				h := r.Range(hMin, hMax)
				x0 := r.Range(0, max(0, BlockInner-w))
				y0 := r.Range(0, max(0, BlockInner-h))
				rects = append(rects, rectI{X0: x0, Y0: y0, X1: x0 + w, Y1: y0 + h})
				continue
			}
			minSpan := max(8, BlockInner/2+r.Range(-2, 2))
			w := r.Range(minSpan, BlockInner)
			h := r.Range(minSpan, BlockInner)
			x0 := r.Range(0, max(0, BlockInner-w)) + r.Range(-2, 2)
			y0 := r.Range(0, max(0, BlockInner-h)) + r.Range(-2, 2)
			x1 := x0 + w + r.Range(-2, 2)
			y1 := y0 + h + r.Range(-2, 2)
			x0 = max(0, x0)
			y0 = max(0, y0)
			x1 = min(BlockInner, x1)
			y1 = min(BlockInner, y1)
			if x1 <= x0 {
				x1 = min(BlockInner, x0+1)
			}
			if y1 <= y0 {
				y1 = min(BlockInner, y0+1)
			}
			rects = append(rects, rectI{X0: x0, Y0: y0, X1: x1, Y1: y1})
		}
		if len(rects) == 0 {
			rects = append(rects, rectI{X0: 0, Y0: 0, X1: BlockInner, Y1: BlockInner})
		}

		treeCount := r.Range(theme.TreeCount[0], theme.TreeCount[1])
		if mega {
			treeCount += r.Range(4, 12)
		}
		if forcedPark && parcelArea > 1 {
			treeCount += 2 + parcelArea
		}
		trees := make([]treeSpec, 0, treeCount)
		for i := 0; i < treeCount; i++ {
			rc := rects[r.Intn(len(rects))]
			big := r.Intn(100) < 14
			rad := 3
			if big {
				rad = r.Range(4, 5)
			}
			pad := rad + 1
			minX := rc.X0 + pad
			maxX := rc.X1 - pad - 1
			minY := rc.Y0 + pad
			maxY := rc.Y1 - pad - 1
			if maxX < minX {
				minX = (rc.X0 + rc.X1) / 2
				maxX = minX
			}
			if maxY < minY {
				minY = (rc.Y0 + rc.Y1) / 2
				maxY = minY
			}
			x := r.Range(minX, maxX)
			y := r.Range(minY, maxY)
			trees = append(trees, treeSpec{X: x, Y: y, Radius: rad})
		}

		return blockFeatures{Seed: seed, Parcel: parcel, IsPark: true, ParkRects: rects, Trees: trees}
	}

	// Building block.
	occ := make([]uint8, BlockInner*BlockInner)
	setOcc := func(x0, y0, x1, y1 int) {
		for y := y0; y < y1; y++ {
			o := y * BlockInner
			for x := x0; x < x1; x++ {
				occ[o+x] = 1
			}
		}
	}
	checkFree := func(x0, y0, x1, y1 int) bool {
		for y := y0; y < y1; y++ {
			o := y * BlockInner
			for x := x0; x < x1; x++ {
				if occ[o+x] != 0 {
					return false
				}
			}
		}
		return true
	}

	buildings := make([]buildingSpec, 0, 8)
	if forcedBuilding {
		profile.SuperBlockChance = max(profile.SuperBlockChance, 76)
		profile.SizeBias += 2
	}
	complex := district > 196 || profile.SuperBlockChance > 44 || forcedBuilding
	target := r.Range(theme.BuildingCount[0], theme.BuildingCount[1]) + profile.BuildingBias
	target = clamp(target, 1, 12)
	if forcedBuilding {
		target = max(target, 2+parcelArea/2)
	}
	margin := 2
	gap := 2
	maxAttempts := 240
	if complex {
		target = max(target, r.Range(theme.BuildingCount[1], theme.BuildingCount[1]+2))
		margin = 1
		gap = 1
		maxAttempts = 480
	}
	if forcedBuilding || (profile.SuperBlockChance > 35 && r.Intn(100) < clamp(profile.SuperBlockChance, 20, 92)) {
		swMin := max(10, BlockInner-8)
		shMin := max(10, BlockInner-8)
		if forcedBuilding {
			swMin = max(12, BlockInner-5)
			shMin = max(12, BlockInner-5)
		}
		sw := r.Range(swMin, BlockInner-2)
		sh := r.Range(shMin, BlockInner-2)
		sx0 := r.Range(1, max(1, BlockInner-1-sw))
		sy0 := r.Range(1, max(1, BlockInner-1-sh))
		base := tp.BuildingA
		switch r.Intn(3) {
		case 1:
			base = tp.BuildingB
		case 2:
			base = tp.BuildingC
		}
		base = base.Add(r.Range(-10, 10), r.Range(-10, 10), r.Range(-10, 10))
		buildings = append(buildings, buildingSpec{
			Col:      base,
			Outline:  base.Mul(190),
			H:        uint8(r.Range(14, 34)),
			Parts:    []rectI{{X0: sx0, Y0: sy0, X1: sx0 + sw, Y1: sy0 + sh}},
			RoofRim:  true,
			RimAddH:  uint8(r.Range(2, 5)),
			UnitAddH: uint8(r.Range(3, 7)),
			HasYard:  !forcedBuilding && r.Intn(100) < 18,
		})
		setOcc(sx0, sy0, sx0+sw, sy0+sh)
		if target > 1 {
			target--
		}
	}

	for attempt := 0; attempt < maxAttempts && len(buildings) < target; attempt++ {
		roll := r.Intn(100)
		shape := "rect"
		switch {
		case roll < 60:
			shape = "rect"
		case roll < 75:
			shape = "L"
		case roll < 90:
			shape = "T"
		default:
			shape = "H"
		}

		minWH := themeSizeToBlock(theme.BuildingSize[0]) + profile.SizeBias
		maxWH := themeSizeToBlock(theme.BuildingSize[1]) + profile.SizeBias
		if complex {
			minWH += 1
			maxWH += 2
		}
		maxDim := BlockInner - 2*margin
		minWH = clamp(minWH, 7, maxDim)
		maxWH = clamp(maxWH, minWH, maxDim)
		w := r.Range(minWH, maxWH)
		h := r.Range(minWH, maxWH)
		if w < 8 || h < 8 {
			continue
		}

		maxX0 := BlockInner - margin - w
		maxY0 := BlockInner - margin - h
		if maxX0 < margin || maxY0 < margin {
			continue
		}
		x0 := r.Range(margin, maxX0)
		y0 := r.Range(margin, maxY0)

		partsLocal := make([]rectI, 0, 4)
		th := r.Range(4, max(5, min(w, h)/2))
		if th*2 > w {
			th = w / 2
		}
		if th*2 > h {
			th = h / 2
		}
		if th < 3 {
			th = 3
		}

		switch shape {
		case "rect":
			partsLocal = append(partsLocal, rectI{X0: 0, Y0: 0, X1: w, Y1: h})
		case "L":
			partsLocal = append(partsLocal,
				rectI{X0: 0, Y0: 0, X1: th, Y1: h},
				rectI{X0: 0, Y0: 0, X1: w, Y1: th},
			)
		case "T":
			cx := w / 2
			xs := cx - th/2
			partsLocal = append(partsLocal,
				rectI{X0: 0, Y0: 0, X1: w, Y1: th},
				rectI{X0: xs, Y0: 0, X1: xs + th, Y1: h},
			)
		case "H":
			ym := h/2 - th/2
			partsLocal = append(partsLocal,
				rectI{X0: 0, Y0: 0, X1: th, Y1: h},
				rectI{X0: w - th, Y0: 0, X1: w, Y1: h},
				rectI{X0: 0, Y0: ym, X1: w, Y1: ym + th},
			)
		}

		ok := true
		for _, pr := range partsLocal {
			gx0 := clamp(x0+pr.X0-gap, 0, BlockInner)
			gy0 := clamp(y0+pr.Y0-gap, 0, BlockInner)
			gx1 := clamp(x0+pr.X1+gap, 0, BlockInner)
			gy1 := clamp(y0+pr.Y1+gap, 0, BlockInner)
			if !checkFree(gx0, gy0, gx1, gy1) {
				ok = false
				break
			}
		}
		if !ok {
			continue
		}

		base := tp.BuildingA
		switch r.Intn(3) {
		case 1:
			base = tp.BuildingB
		case 2:
			base = tp.BuildingC
		}
		base = base.Add(r.Range(-14, 14), r.Range(-14, 14), r.Range(-14, 14))
		outline := base.Mul(200)
		hgt := uint8(r.Range(10, 36))

		parts := make([]rectI, 0, len(partsLocal))
		for _, pr := range partsLocal {
			parts = append(parts, rectI{X0: x0 + pr.X0, Y0: y0 + pr.Y0, X1: x0 + pr.X1, Y1: y0 + pr.Y1})
			setOcc(x0+pr.X0, y0+pr.Y0, x0+pr.X1, y0+pr.Y1)
		}

		b := buildingSpec{
			Col: base, Outline: outline, H: hgt, Parts: parts,
			RoofRim: r.Intn(100) < 75, RimAddH: uint8(r.Range(2, 4)),
			UnitAddH: uint8(r.Range(3, 7)),
			HasYard:  r.Intn(100) < 30, ExtraTrees: r.Intn(100) < 20,
		}

		if r.Intn(100) < 70 {
			unitCount := r.Range(0, 3)
			for ui := 0; ui < unitCount; ui++ {
				uW := r.Range(3, max(3, min(8, w-3)))
				uH := r.Range(3, max(3, min(8, h-3)))
				uX := r.Range(x0+1, max(x0+1, x0+w-uW-1))
				uY := r.Range(y0+1, max(y0+1, y0+h-uH-1))
				b.RoofUnits = append(b.RoofUnits, rectI{X0: uX, Y0: uY, X1: uX + uW, Y1: uY + uH})
			}
		}

		buildings = append(buildings, b)
	}

	lotTreeMax := theme.TreeChance / 3
	if lotTreeMax < 2 {
		lotTreeMax = 2
	}
	lotTreeMax += max(0, profile.ParkBias/10)
	if forcedBuilding {
		lotTreeMax = max(0, lotTreeMax-2)
	}
	treeCount := r.Range(0, lotTreeMax)
	trees := make([]treeSpec, 0, treeCount)
	for i := 0; i < treeCount; i++ {
		for tries := 0; tries < 30; tries++ {
			x := r.Range(4, BlockInner-5)
			y := r.Range(4, BlockInner-5)
			if occ[y*BlockInner+x] == 0 {
				rad := 3
				if r.Intn(100) < 10 {
					rad = r.Range(4, 5)
				}
				trees = append(trees, treeSpec{X: x, Y: y, Radius: rad})
				break
			}
		}
	}

	return blockFeatures{Seed: seed, Parcel: parcel, IsPark: false, Buildings: buildings, Trees: trees}
}

func drawParkingLot(c *Chunk, bx, by int, seed uint64, tp themePalette) {
	baseX, baseY := c.WorldOrigin()
	chunkX0 := baseX
	chunkY0 := baseY
	chunkX1 := baseX + ChunkSize
	chunkY1 := baseY + ChunkSize

	blockX0 := bx*Pattern + (RoadWidth + SidewalkWidth)
	blockY0 := by*Pattern + (RoadWidth + SidewalkWidth)
	blockX1 := blockX0 + BlockInner
	blockY1 := blockY0 + BlockInner
	if blockX0 < BorderThickness || blockY0 < BorderThickness ||
		blockX1 > WorldWidth-BorderThickness || blockY1 > WorldHeight-BorderThickness {
		return
	}

	x0 := clamp(blockX0, chunkX0, chunkX1)
	y0 := clamp(blockY0, chunkY0, chunkY1)
	x1 := clamp(blockX1, chunkX0, chunkX1)
	y1 := clamp(blockY1, chunkY0, chunkY1)
	if x1 <= x0 || y1 <= y0 {
		return
	}

	asphalt := tp.Road.Add(-10, -10, -10)
	asphaltAlt := tp.Road.Add(-4, -4, -3)
	lineCol := roadStripeColor(tp).Add(18, 18, 10)
	for wy := y0; wy < y1; wy++ {
		for wx := x0; wx < x1; wx++ {
			if isPerimeterRoad(wx, wy) {
				continue
			}
			i := c.idx(wx-chunkX0, wy-chunkY0)
			if c.Unbreakable[i] != 0 {
				continue
			}
			if isRoadSurfaceColor(chunkColorAt(c, i), tp) {
				continue
			}
			col := asphalt
			if ((wx + wy) & 3) == 0 {
				col = asphaltAlt
			}
			c.set(i, col, 0, ShadeLit, 0)
		}
	}

	verticalSlots := ((seed >> 60) & 1) == 0
	if verticalSlots {
		aisleY0 := blockY0 + BlockInner/2 - 2
		aisleY1 := aisleY0 + 4
		for wy := max(y0, aisleY0); wy < min(y1, aisleY1); wy++ {
			for wx := x0; wx < x1; wx++ {
				if isPerimeterRoad(wx, wy) {
					continue
				}
				i := c.idx(wx-chunkX0, wy-chunkY0)
				if c.Unbreakable[i] == 0 && !isRoadSurfaceColor(chunkColorAt(c, i), tp) {
					c.set(i, asphaltAlt, 0, ShadeLit, 0)
				}
			}
		}
		for lx := 3; lx < BlockInner-3; lx += 4 {
			wx := blockX0 + lx
			if wx < x0 || wx >= x1 {
				continue
			}
			for ly := 2; ly < BlockInner-2; ly++ {
				if ly >= BlockInner/2-2 && ly < BlockInner/2+2 {
					continue
				}
				if ly%7 == 0 {
					continue
				}
				wy := blockY0 + ly
				if wy < y0 || wy >= y1 {
					continue
				}
				i := c.idx(wx-chunkX0, wy-chunkY0)
				if c.Unbreakable[i] == 0 && !isRoadSurfaceColor(chunkColorAt(c, i), tp) {
					c.set(i, lineCol, 0, ShadeLit, 0)
				}
			}
		}
	} else {
		aisleX0 := blockX0 + BlockInner/2 - 2
		aisleX1 := aisleX0 + 4
		for wx := max(x0, aisleX0); wx < min(x1, aisleX1); wx++ {
			for wy := y0; wy < y1; wy++ {
				if isPerimeterRoad(wx, wy) {
					continue
				}
				i := c.idx(wx-chunkX0, wy-chunkY0)
				if c.Unbreakable[i] == 0 && !isRoadSurfaceColor(chunkColorAt(c, i), tp) {
					c.set(i, asphaltAlt, 0, ShadeLit, 0)
				}
			}
		}
		for ly := 3; ly < BlockInner-3; ly += 4 {
			wy := blockY0 + ly
			if wy < y0 || wy >= y1 {
				continue
			}
			for lx := 2; lx < BlockInner-2; lx++ {
				if lx >= BlockInner/2-2 && lx < BlockInner/2+2 {
					continue
				}
				if lx%7 == 0 {
					continue
				}
				wx := blockX0 + lx
				if wx < x0 || wx >= x1 {
					continue
				}
				i := c.idx(wx-chunkX0, wy-chunkY0)
				if c.Unbreakable[i] == 0 && !isRoadSurfaceColor(chunkColorAt(c, i), tp) {
					c.set(i, lineCol, 0, ShadeLit, 0)
				}
			}
		}
	}
}

func parcelInnerBounds(p blockParcel) (x0, y0, x1, y1 int) {
	x0 = p.AX*Pattern + (RoadWidth + SidewalkWidth)
	y0 = p.AY*Pattern + (RoadWidth + SidewalkWidth)
	x1 = (p.AX+p.W-1)*Pattern + (RoadWidth + SidewalkWidth + BlockInner)
	y1 = (p.AY+p.H-1)*Pattern + (RoadWidth + SidewalkWidth + BlockInner)
	return
}

func drawMergedParkParcel(c *Chunk, p blockParcel, seed uint64, tp themePalette, theme ThemeConfig) {
	if p.W <= 1 && p.H <= 1 {
		return
	}
	x0, y0, x1, y1 := parcelInnerBounds(p)
	if x0 < BorderThickness || y0 < BorderThickness || x1 > WorldWidth-BorderThickness || y1 > WorldHeight-BorderThickness {
		return
	}

	baseX, baseY := c.WorldOrigin()
	chunkX0 := baseX
	chunkY0 := baseY
	chunkX1 := baseX + ChunkSize
	chunkY1 := baseY + ChunkSize

	fx0 := clamp(x0, chunkX0, chunkX1)
	fy0 := clamp(y0, chunkY0, chunkY1)
	fx1 := clamp(x1, chunkX0, chunkX1)
	fy1 := clamp(y1, chunkY0, chunkY1)
	for wy := fy0; wy < fy1; wy++ {
		for wx := fx0; wx < fx1; wx++ {
			if isPerimeterRoad(wx, wy) {
				continue
			}
			i := c.idx(wx-chunkX0, wy-chunkY0)
			if c.Unbreakable[i] != 0 {
				continue
			}
			if isRoadSurfaceColor(chunkColorAt(c, i), tp) {
				continue
			}
			pv := uint8(hash2D(seed^0xBADC0FFEE0DDF00D, wx>>4, wy>>4) >> 56)
			col := tp.Grass
			if pv < 22 {
				col = tp.GrassTorn
			} else if pv < 72 {
				col = tp.GrassPatch
			}
			hv := uint8(hash2D(seed^0x3344AA77, wx, wy) >> 56)
			switch (wx + 2*wy + int(hv&3)) & 3 {
			case 1:
				col = col.Add(6, 6, 4)
			case 2:
				col = col.Add(-6, -6, -4)
			}
			c.set(i, col, 0, ShadeLit, 0)
		}
	}

	r := NewRand(seed ^ 0xCAFED00D)
	area := p.W * p.H
	treeMin := max(2, theme.TreeCount[0]*area/2)
	treeMax := max(treeMin+2, theme.TreeCount[1]*area/2+area*2)
	for t := 0; t < r.Range(treeMin, treeMax); t++ {
		tx := r.Range(x0+4, x1-5)
		ty := r.Range(y0+4, y1-5)
		if tx < chunkX0 || tx >= chunkX1 || ty < chunkY0 || ty >= chunkY1 {
			continue
		}
		i := c.idx(tx-chunkX0, ty-chunkY0)
		if c.Unbreakable[i] != 0 || c.Height[i] != 0 {
			continue
		}
		if isRoadSurfaceColor(chunkColorAt(c, i), tp) {
			continue
		}
		drawTreeSized(c, tx, ty, r.Range(3, 5), tp, isWinterTheme(theme))
	}
}

func drawMergedBuildingParcel(c *Chunk, p blockParcel, seed uint64, tp themePalette, theme ThemeConfig) {
	if p.W <= 1 && p.H <= 1 {
		return
	}
	x0, y0, x1, y1 := parcelInnerBounds(p)
	if x0 < BorderThickness || y0 < BorderThickness || x1 > WorldWidth-BorderThickness || y1 > WorldHeight-BorderThickness {
		return
	}

	baseX, baseY := c.WorldOrigin()
	chunkX0 := baseX
	chunkY0 := baseY
	chunkX1 := baseX + ChunkSize
	chunkY1 := baseY + ChunkSize

	fx0 := clamp(x0, chunkX0, chunkX1)
	fy0 := clamp(y0, chunkY0, chunkY1)
	fx1 := clamp(x1, chunkX0, chunkX1)
	fy1 := clamp(y1, chunkY0, chunkY1)
	for wy := fy0; wy < fy1; wy++ {
		for wx := fx0; wx < fx1; wx++ {
			if isPerimeterRoad(wx, wy) {
				continue
			}
			i := c.idx(wx-chunkX0, wy-chunkY0)
			if c.Unbreakable[i] != 0 || c.Height[i] != 0 {
				continue
			}
			if isRoadSurfaceColor(chunkColorAt(c, i), tp) {
				continue
			}
			col := tp.Lot
			if ((wx + wy) & 3) == 0 {
				col = tp.Lot.Add(4, 4, 2)
			}
			c.set(i, col, 0, ShadeLit, 0)
		}
	}

	width := x1 - x0
	height := y1 - y0
	if width < 12 || height < 12 {
		return
	}
	occ := make([]uint8, width*height)
	setOcc := func(rx0, ry0, rx1, ry1 int) {
		for y := ry0; y < ry1; y++ {
			o := y * width
			for x := rx0; x < rx1; x++ {
				occ[o+x] = 1
			}
		}
	}
	checkFree := func(rx0, ry0, rx1, ry1 int) bool {
		if rx0 < 0 || ry0 < 0 || rx1 > width || ry1 > height {
			return false
		}
		for y := ry0; y < ry1; y++ {
			o := y * width
			for x := rx0; x < rx1; x++ {
				if occ[o+x] != 0 {
					return false
				}
			}
		}
		return true
	}

	r := NewRand(seed ^ 0x8E71AA44)
	winterTheme := isWinterTheme(theme)
	area := p.W * p.H
	target := 2 + area*2 + r.Range(0, area)
	for attempt := 0; attempt < 480 && target > 0; attempt++ {
		w := r.Range(max(8, width/6), max(12, width/2))
		h := r.Range(max(8, height/6), max(12, height/2))
		if w >= width-3 || h >= height-3 {
			continue
		}
		rx0 := r.Range(1, width-w-2)
		ry0 := r.Range(1, height-h-2)
		gap := 2
		if !checkFree(rx0-gap, ry0-gap, rx0+w+gap, ry0+h+gap) {
			continue
		}
		setOcc(rx0, ry0, rx0+w, ry0+h)
		target--

		base := tp.BuildingA
		switch r.Intn(3) {
		case 1:
			base = tp.BuildingB
		case 2:
			base = tp.BuildingC
		}
		base = base.Add(r.Range(-10, 10), r.Range(-10, 10), r.Range(-10, 10))
		outline := base.Mul(190)
		hgt := uint8(r.Range(12, 36))

		wx0 := x0 + rx0
		wy0 := y0 + ry0
		wx1 := wx0 + w
		wy1 := wy0 + h
		dx0 := clamp(wx0, chunkX0, chunkX1)
		dy0 := clamp(wy0, chunkY0, chunkY1)
		dx1 := clamp(wx1, chunkX0, chunkX1)
		dy1 := clamp(wy1, chunkY0, chunkY1)
		for wy := dy0; wy < dy1; wy++ {
			for wx := dx0; wx < dx1; wx++ {
				if isPerimeterRoad(wx, wy) {
					continue
				}
				i := c.idx(wx-chunkX0, wy-chunkY0)
				if c.Unbreakable[i] != 0 {
					continue
				}
				if isRoadSurfaceColor(chunkColorAt(c, i), tp) {
					continue
				}
				edge := wx == wx0 || wx == wx1-1 || wy == wy0 || wy == wy1-1
				col := base
				ch := hgt
				if edge {
					col = outline
					ch = hgt + 2
				}
				if winterTheme {
					hv := uint8(hash2D(seed^0xD1CEFACE, wx, wy) >> 56)
					if edge {
						if hv < 96 {
							col = col.Add(34, 38, 42)
							ch++
						}
					} else if hv < 228 {
						col = col.Add(72, 78, 84)
					}
				}
				c.set(i, col, ch, ShadeLit, 0)
			}
		}
	}
}

func applyBlockFeatures(c *Chunk, bx, by int, feat blockFeatures, tp themePalette, theme ThemeConfig) {
	if feat.IsParking {
		drawParkingLot(c, bx, by, feat.Seed, tp)
		return
	}
	if feat.Parcel.Kind != parcelNone && feat.Parcel.W*feat.Parcel.H > 1 {
		if feat.Parcel.Kind == parcelPark {
			drawMergedParkParcel(c, feat.Parcel, feat.Seed, tp, theme)
		} else {
			drawMergedBuildingParcel(c, feat.Parcel, feat.Seed, tp, theme)
		}
		return
	}

	baseX, baseY := c.WorldOrigin()
	chunkX0 := baseX
	chunkY0 := baseY
	chunkX1 := baseX + ChunkSize
	chunkY1 := baseY + ChunkSize

	blockX0 := bx*Pattern + (RoadWidth + SidewalkWidth)
	blockY0 := by*Pattern + (RoadWidth + SidewalkWidth)
	blockX1 := blockX0 + BlockInner
	blockY1 := blockY0 + BlockInner

	// Avoid partial edge lots; they produce cut-off buildings/parks at world borders.
	if blockX0 < BorderThickness || blockY0 < BorderThickness ||
		blockX1 > WorldWidth-BorderThickness || blockY1 > WorldHeight-BorderThickness {
		return
	}

	if blockX1 <= chunkX0 || blockX0 >= chunkX1 || blockY1 <= chunkY0 || blockY0 >= chunkY1 {
		return
	}

	// Parks.
	if feat.IsPark {
		for _, pr := range feat.ParkRects {
			rx0 := blockX0 + pr.X0
			ry0 := blockY0 + pr.Y0
			rx1 := blockX0 + pr.X1
			ry1 := blockY0 + pr.Y1
			x0 := clamp(rx0, chunkX0, chunkX1)
			y0 := clamp(ry0, chunkY0, chunkY1)
			x1 := clamp(rx1, chunkX0, chunkX1)
			y1 := clamp(ry1, chunkY0, chunkY1)
			for wy := y0; wy < y1; wy++ {
				ly := wy - chunkY0
				for wx := x0; wx < x1; wx++ {
					if isPerimeterRoad(wx, wy) {
						continue
					}
					lx := wx - chunkX0
					i := c.idx(lx, ly)
					if c.Unbreakable[i] != 0 {
						continue
					}
					dx := min(wx-(blockX0+pr.X0), (blockX0+pr.X1)-1-wx)
					dy := min(wy-(blockY0+pr.Y0), (blockY0+pr.Y1)-1-wy)
					edgeDist := min(dx, dy)
					nv := int(int8(hash2D(feat.Seed^0xCAFEBABE, wx, wy) >> 56))
					noise := nv % 7
					threshold := 4 + int((hash2D(feat.Seed^0xDEADBEEF, wx>>3, wy>>3)&0xFF)>>6)
					if edgeDist+noise >= threshold {
						col := tp.Grass
						pv := uint8(hash2D(feat.Seed^0xBADC0FFEE0DDF00D, wx>>4, wy>>4) >> 56)
						if pv < 18 {
							col = tp.GrassTorn
						} else if pv < 60 {
							col = tp.GrassPatch
						}
						hv := uint8(hash2D(feat.Seed^0x1234567, wx, wy) >> 56)
						if pv < 22 && (hv&7) == 0 {
							col = tp.Lot.Add(6, 4, 2)
						}
						switch (wx + 2*wy + int(hv&3)) & 3 {
						case 1:
							col = col.Add(6, 6, 4)
						case 2:
							col = col.Add(-6, -6, -4)
						}
						c.set(i, col, 0, ShadeLit, 0)
					} else if edgeDist+noise >= threshold-2 {
						hv := uint8(hash2D(feat.Seed^0x33445566, wx, wy) >> 56)
						if (hv & 3) == 0 {
							col := tp.GrassPatch.Add(-8, -8, -6)
							c.set(i, col, 0, ShadeLit, 0)
						}
					}
				}
			}
		}
	}

	// Buildings.
	winterTheme := isWinterTheme(theme)
	for _, b := range feat.Buildings {
		mask := make([]uint8, BlockInner*BlockInner)
		minLX, minLY := BlockInner, BlockInner
		maxLX, maxLY := 0, 0
		for _, pr := range b.Parts {
			if pr.X0 < minLX {
				minLX = pr.X0
			}
			if pr.Y0 < minLY {
				minLY = pr.Y0
			}
			if pr.X1 > maxLX {
				maxLX = pr.X1
			}
			if pr.Y1 > maxLY {
				maxLY = pr.Y1
			}
			for y := pr.Y0; y < pr.Y1; y++ {
				o := y * BlockInner
				for x := pr.X0; x < pr.X1; x++ {
					mask[o+x] = 1
				}
			}
		}
		if minLX >= maxLX || minLY >= maxLY {
			continue
		}

		rx0 := blockX0 + minLX
		ry0 := blockY0 + minLY
		rx1 := blockX0 + maxLX
		ry1 := blockY0 + maxLY

		// Keep the outer ring road clear and avoid edge-clipped buildings.
		if rx0 < BorderThickness+RoadWidth ||
			ry0 < BorderThickness+RoadWidth ||
			rx1 > WorldWidth-BorderThickness-RoadWidth ||
			ry1 > WorldHeight-BorderThickness-RoadWidth {
			continue
		}

		x0 := clamp(rx0, chunkX0, chunkX1)
		y0 := clamp(ry0, chunkY0, chunkY1)
		x1 := clamp(rx1, chunkX0, chunkX1)
		y1 := clamp(ry1, chunkY0, chunkY1)
		if x1 <= x0 || y1 <= y0 {
			continue
		}

		for wy := y0; wy < y1; wy++ {
			lyChunk := wy - chunkY0
			lyBlock := wy - blockY0
			if lyBlock < 0 || lyBlock >= BlockInner {
				continue
			}
			rowOff := lyBlock * BlockInner
			for wx := x0; wx < x1; wx++ {
				if isPerimeterRoad(wx, wy) {
					continue
				}
				lxChunk := wx - chunkX0
				lxBlock := wx - blockX0
				if lxBlock < 0 || lxBlock >= BlockInner {
					continue
				}
				if mask[rowOff+lxBlock] == 0 {
					continue
				}
				i := c.idx(lxChunk, lyChunk)
				if c.Unbreakable[i] != 0 {
					continue
				}

				isEdge := false
				for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					nx := lxBlock + d[0]
					ny := lyBlock + d[1]
					if nx < 0 || ny < 0 || nx >= BlockInner || ny >= BlockInner {
						isEdge = true
						break
					}
					if mask[ny*BlockInner+nx] == 0 {
						isEdge = true
						break
					}
				}

				col := b.Col
				hgt := b.H
				if isEdge {
					col = b.Outline
					if b.RoofRim {
						hgt = b.H + b.RimAddH
					}
				}

				// Grassy yards around building edges.
				if isEdge {
					ringWidth := 3
					if b.HasYard {
						ringWidth = 6
					}
					for ry := -ringWidth; ry <= ringWidth; ry++ {
						for rx := -ringWidth; rx <= ringWidth; rx++ {
							if rx == 0 && ry == 0 {
								continue
							}
							nDist := abs(rx)
							if abs(ry) > nDist {
								nDist = abs(ry)
							}
							nx := lxBlock + rx
							ny := lyBlock + ry
							if nx < 0 || ny < 0 || nx >= BlockInner || ny >= BlockInner {
								continue
							}
							if mask[ny*BlockInner+nx] != 0 {
								continue
							}
							wxn := blockX0 + nx
							wyn := blockY0 + ny
							if wxn < chunkX0 || wxn >= chunkX1 || wyn < chunkY0 || wyn >= chunkY1 {
								continue
							}
							if isPerimeterRoad(wxn, wyn) {
								continue
							}
							lxChunkN := wxn - chunkX0
							lyChunkN := wyn - chunkY0
							curIdx := c.idx(lxChunkN, lyChunkN)
							curOff := c.pixOff(curIdx)
							curCol := RGB{R: c.Pixels[curOff+0], G: c.Pixels[curOff+1], B: c.Pixels[curOff+2]}
							if !rgbEq(curCol, tp.Lot) {
								continue
							}
							hv := uint8(hash2D(feat.Seed^0xABCDEF01, wxn, wyn) >> 56)
							pv := int(hv & 7)
							if nDist < 2 {
								continue
							}
							effectiveDist := nDist
							if b.HasYard {
								effectiveDist = nDist + 1
							}
							if pv < effectiveDist {
								gcol := tp.GrassPatch
								if (hv & 7) == 0 {
									gcol = tp.Grass
								}
								c.set(curIdx, gcol, 0, ShadeLit, 0)
							}
						}
					}
				}

				// Roof units (cooling units).
				for _, u := range b.RoofUnits {
					if lxBlock >= u.X0 && lxBlock < u.X1 && lyBlock >= u.Y0 && lyBlock < u.Y1 {
						col = tp.BuildingDark
						hgt = b.H + b.UnitAddH
						if lxBlock == u.X0 || lxBlock == u.X1-1 || lyBlock == u.Y0 || lyBlock == u.Y1-1 {
							col = tp.BuildingDark.Mul(180)
							hgt = b.H + b.UnitAddH + 1
						}
						break
					}
				}
				if winterTheme {
					hv := uint8(hash2D(feat.Seed^0xA11CE5ED, wx, wy) >> 56)
					if isEdge {
						if hv < 104 {
							col = col.Add(38, 42, 46)
							hgt++
						}
					} else if hv < 228 {
						col = col.Add(72, 78, 84)
					}
				}

				c.set(i, col, hgt, ShadeLit, 0)
			}
		}
	}

	// Block-level grass patches.
	if len(feat.Buildings) > 0 {
		r := NewRand(feat.Seed ^ 0xFEEDFACE)
		patchCount := r.Range(1, 4)
		for p := 0; p < patchCount; p++ {
			cx := r.Range(4, BlockInner-5)
			cy := r.Range(4, BlockInner-5)
			rad := r.Range(2, 5)
			for dy := -rad; dy <= rad; dy++ {
				for dx := -rad; dx <= rad; dx++ {
					x := cx + dx
					y := cy + dy
					if x < 0 || y < 0 || x >= BlockInner || y >= BlockInner {
						continue
					}
					overlap := false
					for _, bb := range feat.Buildings {
						for _, pr := range bb.Parts {
							if x >= pr.X0 && x < pr.X1 && y >= pr.Y0 && y < pr.Y1 {
								overlap = true
								break
							}
						}
						if overlap {
							break
						}
					}
					if overlap {
						continue
					}
					d := abs(dx)
					if abs(dy) > d {
						d = abs(dy)
					}
					if d > rad {
						continue
					}
					wx := blockX0 + x
					wy := blockY0 + y
					if wx < chunkX0 || wx >= chunkX1 || wy < chunkY0 || wy >= chunkY1 {
						continue
					}
					if isPerimeterRoad(wx, wy) {
						continue
					}
					ci := c.idx(wx-chunkX0, wy-chunkY0)
					if c.Unbreakable[ci] != 0 {
						continue
					}
					if r.Intn(rad+2) >= d {
						col := tp.GrassPatch
						if r.Intn(8) == 0 {
							col = tp.Grass
						}
						c.set(ci, col, 0, ShadeLit, 0)
					}
				}
			}
			if r.Intn(100) < 25 {
				tx := clamp(cx+r.Range(-rad, rad), 0, BlockInner-1)
				ty := clamp(cy+r.Range(-rad, rad), 0, BlockInner-1)
				wx := blockX0 + tx
				wy := blockY0 + ty
				if wx >= chunkX0 && wx < chunkX1 && wy >= chunkY0 && wy < chunkY1 {
					if isPerimeterRoad(wx, wy) {
						continue
					}
					ci := c.idx(wx-chunkX0, wy-chunkY0)
					if c.Unbreakable[ci] == 0 {
						drawTreeSized(c, wx, wy, r.Range(3, 5), tp, winterTheme)
					}
				}
			}
		}
	}

	// Trees.
	for _, t := range feat.Trees {
		r := t.Radius
		if r < 3 {
			r = 3
		}
		if r > 6 {
			r = 6
		}
		drawTreeSized(c, blockX0+t.X, blockY0+t.Y, r, tp, winterTheme)
	}
}

// drawTreeSized draws a layered circular canopy.
func drawTreeSized(c *Chunk, wx, wy int, radius int, tp themePalette, winter bool) {
	baseX, baseY := c.WorldOrigin()
	chunkX0 := baseX
	chunkY0 := baseY
	chunkX1 := baseX + ChunkSize
	chunkY1 := baseY + ChunkSize

	type layer struct {
		r   int
		col RGB
		h   uint8
	}
	base := radius
	mid := radius - 1
	top := radius - 2
	if mid < 2 {
		mid = 2
	}
	if top < 1 {
		top = 1
	}
	h0 := uint8(11 + radius)
	layers := []layer{
		{r: base, col: tp.TreeBase, h: h0},
		{r: mid, col: tp.TreeMid, h: h0 + 2},
		{r: top, col: tp.TreeTop, h: h0 + 4},
	}

	shadowSideX := -SunDx
	shadowSideY := -SunDy

	treeSeed := uint64(wx)*0x9E3779B97F4A7C15 ^ uint64(wy)*0xC2B2AE3D27D4EB4F
	rgen := NewRand(hash2D(treeSeed, wx, wy))
	cxJitter := rgen.Range(-1, 0)
	cyJitter := rgen.Range(-1, 0)

	for _, L := range layers {
		baseR := L.r
		rJ := rgen.Range(-1, 0)
		localR := baseR + rJ
		if localR < 1 {
			localR = 1
		}
		for dy := -localR; dy <= localR; dy++ {
			span := int(math.Floor(math.Sqrt(float64(localR*localR - dy*dy))))
			span += rgen.Range(-1, 0)
			if span < 0 {
				span = 0
			}
			for dx := -span; dx <= span; dx++ {
				tx := wx + dx + cxJitter
				ty := wy + dy + cyJitter
				if tx < chunkX0 || tx >= chunkX1 || ty < chunkY0 || ty >= chunkY1 {
					continue
				}
				lx := tx - chunkX0
				ly := ty - chunkY0
				i := c.idx(lx, ly)
				if c.Unbreakable[i] != 0 {
					continue
				}
				if isPerimeterRoad(tx, ty) {
					continue
				}
				if isRoadSurfaceColor(chunkColorAt(c, i), tp) {
					continue
				}
				col := L.col
				if dx*shadowSideX+dy*shadowSideY > 0 {
					col = col.Mul(210)
				}
				if winter {
					hv := uint8(hash2D(treeSeed^0x51D4A7E0, tx, ty) >> 56)
					if L.h >= h0+4 {
						if hv < 234 {
							col = col.Add(86, 92, 96)
						}
					} else if L.h >= h0+2 {
						if hv < 180 {
							col = col.Add(46, 52, 56)
						}
					}
				}
				c.set(i, col, L.h, ShadeLit, 0)
			}
		}
	}
}
