package game

// RectF is an axis-aligned rectangle in world-pixel space.
type RectF struct {
	X0, Y0 float64
	X1, Y1 float64
}

func (r RectF) Intersects(o RectF) bool {
	return r.X0 < o.X1 && r.X1 > o.X0 && r.Y0 < o.Y1 && r.Y1 > o.Y0
}

func (r RectF) Contains(o RectF) bool {
	return o.X0 >= r.X0 && o.X1 <= r.X1 && o.Y0 >= r.Y0 && o.Y1 <= r.Y1
}

type ChunkKey struct {
	X, Y int
}

type quadItem struct {
	key    ChunkKey
	bounds RectF
}

// QuadNode is a simple quadtree for frustum culling of chunks.
type QuadNode struct {
	bounds RectF
	depth  int
	items  []quadItem
	child  [4]*QuadNode
}

func NewQuadNode(bounds RectF, depth int) *QuadNode {
	return &QuadNode{
		bounds: bounds,
		depth:  depth,
		items:  make([]quadItem, 0, QuadCapacity),
	}
}

func (n *QuadNode) Insert(key ChunkKey, bounds RectF) {
	if n.child[0] != nil {
		if c := n.childThatContains(bounds); c != nil {
			c.Insert(key, bounds)
			return
		}
	}

	n.items = append(n.items, quadItem{key: key, bounds: bounds})

	if len(n.items) > QuadCapacity && n.depth < QuadMaxDepth {
		n.subdivide()
		kept := n.items[:0]
		for _, it := range n.items {
			if c := n.childThatContains(it.bounds); c != nil {
				c.Insert(it.key, it.bounds)
			} else {
				kept = append(kept, it)
			}
		}
		n.items = kept
	}
}

func (n *QuadNode) Query(r RectF, out *[]ChunkKey) {
	if !n.bounds.Intersects(r) {
		return
	}
	for _, it := range n.items {
		if it.bounds.Intersects(r) {
			*out = append(*out, it.key)
		}
	}
	if n.child[0] == nil {
		return
	}
	for i := 0; i < 4; i++ {
		if n.child[i] != nil {
			n.child[i].Query(r, out)
		}
	}
}

func (n *QuadNode) subdivide() {
	if n.child[0] != nil {
		return
	}
	mx := (n.bounds.X0 + n.bounds.X1) * 0.5
	my := (n.bounds.Y0 + n.bounds.Y1) * 0.5
	n.child[0] = NewQuadNode(RectF{X0: n.bounds.X0, Y0: n.bounds.Y0, X1: mx, Y1: my}, n.depth+1)
	n.child[1] = NewQuadNode(RectF{X0: mx, Y0: n.bounds.Y0, X1: n.bounds.X1, Y1: my}, n.depth+1)
	n.child[2] = NewQuadNode(RectF{X0: n.bounds.X0, Y0: my, X1: mx, Y1: n.bounds.Y1}, n.depth+1)
	n.child[3] = NewQuadNode(RectF{X0: mx, Y0: my, X1: n.bounds.X1, Y1: n.bounds.Y1}, n.depth+1)
}

func (n *QuadNode) childThatContains(b RectF) *QuadNode {
	for i := 0; i < 4; i++ {
		c := n.child[i]
		if c != nil && c.bounds.Contains(b) {
			return c
		}
	}
	return nil
}
